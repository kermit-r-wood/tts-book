import React, { useState, useRef, useEffect } from 'react';
import { Folder, Globe, ChevronDown, Play, Square } from 'lucide-react';
import { api } from '../services/api';

const emotionMap = {
    'calm': '平静',
    'happy': '开心',
    'angry': '生气',
    'sad': '悲伤',
    'afraid': '恐惧',
    'disgusted': '厌恶',
    'melancholic': '忧郁',
    'surprised': '惊讶'
};

export default function CharacterRow({ char, mappingData, updateMapping, voiceOptions, handleLocalFileSelect, selected, onSelect }) {
    const [isOpen, setIsOpen] = useState(false); // Controls dropdown open/close
    const [playingUrl, setPlayingUrl] = useState(null); // Track which URL is currently playing
    const audioRef = useRef(null); // Keep track of the Audio object
    const dropdownRef = useRef(null);

    // Close dropdown when clicking outside
    useEffect(() => {
        function handleClickOutside(event) {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
                setIsOpen(false);
            }
        }
        document.addEventListener("mousedown", handleClickOutside);
        return () => {
            document.removeEventListener("mousedown", handleClickOutside);
        };
    }, [dropdownRef]);

    // Cleanup audio on unmount
    useEffect(() => {
        return () => {
            if (audioRef.current) {
                audioRef.current.pause();
                audioRef.current = null;
            }
        };
    }, []);

    const handlePlay = (e, path) => {
        e.stopPropagation(); // Prevent closing dropdown or selecting
        const url = api.getVoicePreviewUrl(path);

        // If clicking the same one that is playing, stop it
        if (playingUrl === url) {
            if (audioRef.current) {
                audioRef.current.pause();
                audioRef.current = null;
            }
            setPlayingUrl(null);
            return;
        }

        // Stop existing
        if (audioRef.current) {
            audioRef.current.pause();
        }

        // Play new
        const audio = new Audio(url);
        audioRef.current = audio;
        setPlayingUrl(url);

        audio.play().catch(err => {
            console.error("Failed to play audio:", err);
            setPlayingUrl(null);
        });

        audio.onended = () => {
            setPlayingUrl(null);
            audioRef.current = null;
        };
    };

    const selectedOption = voiceOptions.find(v => v.path === mappingData?.voiceId);
    const getCleanName = (name) => name.replace(/\s*\(\)\s*$/, '');
    const displayValue = selectedOption ? getCleanName(selectedOption.name) : (mappingData?.voiceId || '');

    return (
        <tr className={`border-b border-gray-700/50 transition-colors ${selected ? 'bg-violet-900/10' : 'hover:bg-white/5'}`}>
            <td className="p-3 w-12">
                <input
                    type="checkbox"
                    checked={selected || false}
                    onChange={(e) => onSelect && onSelect(char, e.target.checked)}
                    className="rounded border-gray-600 bg-gray-700 text-violet-500 focus:ring-violet-500 focus:ring-offset-gray-800"
                />
            </td>
            <td className="p-3 font-medium text-violet-300">
                {char}
            </td>
            <td className="p-3">
                <div className="flex gap-2 items-center">
                    <div className="flex-1 relative" ref={dropdownRef}>
                        <div className="relative">
                            <input
                                className="input-field text-sm py-1 w-full pl-2 pr-8"
                                placeholder="选择或输入语音路径"
                                value={displayValue}
                                onChange={e => {
                                    // If user types, we assume they are entering a custom path or breaking the selection
                                    // This is a simple controlled input where value is bound to the mapping
                                    // But since we visually show NAME for known ID, typing breaks that link immediately unless we search back.
                                    // For now, direct update.
                                    updateMapping(char, 'voiceId', e.target.value)
                                }}
                                onClick={() => setIsOpen(!isOpen)}
                                title={mappingData?.voiceId}
                            />
                            {voiceOptions.length > 0 && (
                                <div
                                    className="absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer text-gray-400 hover:text-white"
                                    onClick={() => setIsOpen(!isOpen)}
                                >
                                    <ChevronDown size={14} />
                                </div>
                            )}
                        </div>

                        {/* Custom Dropdown */}
                        {isOpen && voiceOptions.length > 0 && (
                            <div className="absolute top-full left-0 right-0 mt-1 z-50 glass-panel max-h-60 overflow-y-auto border border-gray-600 shadow-xl">
                                {voiceOptions.map(opt => {
                                    const previewUrl = api.getVoicePreviewUrl(opt.path);
                                    const isPlaying = playingUrl === previewUrl;
                                    const diffName = getCleanName(opt.name);

                                    return (
                                        <div
                                            key={opt.path}
                                            className="flex items-center justify-between px-3 py-2 text-sm hover:bg-violet-600/50 cursor-pointer text-gray-200 transition-colors"
                                            onClick={() => {
                                                updateMapping(char, 'voiceId', opt.path);
                                                setIsOpen(false);
                                            }}
                                        >
                                            <div className="truncate flex-1 mr-2" title={opt.path}>
                                                {diffName}
                                            </div>
                                            <button
                                                className="text-gray-400 hover:text-white p-1 rounded hover:bg-white/10"
                                                onClick={(e) => handlePlay(e, opt.path)}
                                                title={isPlaying ? "停止试听" : "试听"}
                                            >
                                                {isPlaying ? <Square size={14} fill="currentColor" /> : <Play size={14} fill="currentColor" />}
                                            </button>
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </div>

                    <div className="flex gap-1">
                        <label className="btn-secondary px-2 py-1 text-xs flex items-center gap-1 cursor-pointer" title="选择本地文件">
                            <Globe size={14} />
                            <input
                                type="file"
                                className="hidden"
                                accept="audio/*"
                                onChange={(e) => handleLocalFileSelect(char, e)}
                            />
                        </label>
                    </div>
                </div>
            </td>
            <td className="p-3">
                <select
                    className="input-field w-full text-sm py-1 bg-slate-800 cursor-pointer relative z-10"
                    value={mappingData?.emotion || 'calm'}
                    onChange={e => updateMapping(char, 'emotion', e.target.value)}
                >
                    {Object.keys(emotionMap).map(em => (
                        <option key={em} value={em}>{emotionMap[em]}</option>
                    ))}
                </select>
            </td>
            <td className="p-3">
                <input
                    type="number" step="0.1"
                    className="input-field text-sm py-1 w-20"
                    placeholder="1.0"
                    value={mappingData?.speed ?? 1.0}
                    onChange={e => {
                        const val = e.target.value;
                        // Allow empty or partial input during typing
                        if (val === '' || val === '-' || val === '.' || val.endsWith('.')) {
                            updateMapping(char, 'speed', val);
                        } else {
                            const num = parseFloat(val);
                            if (!isNaN(num)) {
                                updateMapping(char, 'speed', num);
                            }
                        }
                    }}
                    onBlur={e => {
                        // On blur, ensure we have a valid number
                        const val = e.target.value;
                        const num = parseFloat(val);
                        if (isNaN(num) || val === '') {
                            updateMapping(char, 'speed', 1.0);
                        } else {
                            updateMapping(char, 'speed', num);
                        }
                    }}
                />
            </td>
            <td className="p-3">
                <button
                    onClick={() => updateMapping(char, 'useLLMEmotion', !(mappingData?.useLLMEmotion ?? true))}
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all ${(mappingData?.useLLMEmotion ?? true)
                        ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30 hover:bg-violet-600/30'
                        : 'bg-gray-700/50 text-gray-400 border border-gray-600/50 hover:bg-gray-700'
                        }`}
                >
                    {(mappingData?.useLLMEmotion ?? true) ? 'LLM分析' : '默认情感'}
                </button>
            </td>
        </tr>
    );
}
