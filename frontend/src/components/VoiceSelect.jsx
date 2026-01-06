import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, Play, Square } from 'lucide-react';
import { api } from '../services/api';

export default function VoiceSelect({ value, onChange, options, placeholder = "Select voice" }) {
    const [isOpen, setIsOpen] = useState(false);
    const [playingUrl, setPlayingUrl] = useState(null);
    const audioRef = useRef(null);
    const dropdownRef = useRef(null);

    // Close dropdown on outside click
    useEffect(() => {
        function handleClickOutside(event) {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
                setIsOpen(false);
            }
        }
        document.addEventListener("mousedown", handleClickOutside);
        return () => document.removeEventListener("mousedown", handleClickOutside);
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
        e.stopPropagation();
        const url = api.getVoicePreviewUrl(path);

        if (playingUrl === url) {
            if (audioRef.current) {
                audioRef.current.pause();
                audioRef.current = null;
            }
            setPlayingUrl(null);
            return;
        }

        if (audioRef.current) {
            audioRef.current.pause();
        }

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

    const selectedOption = options.find(v => v.path === value || v.id === value);
    const getCleanName = (name) => name.replace(/\s*\(\)\s*$/, '');
    const displayValue = selectedOption ? getCleanName(selectedOption.name) : (value || '');

    return (
        <div className="relative w-full" ref={dropdownRef}>
            <div className="relative">
                <input
                    className="input-field text-sm py-1.5 w-full pl-3 pr-8 bg-slate-800 border-gray-600 focus:border-violet-500 rounded text-gray-200 outline-none"
                    placeholder={placeholder}
                    value={displayValue}
                    onChange={e => onChange(e.target.value)}
                    onClick={() => setIsOpen(!isOpen)}
                    title={value}
                />
                <div
                    className="absolute right-2 top-1/2 -translate-y-1/2 cursor-pointer text-gray-400 hover:text-white"
                    onClick={() => setIsOpen(!isOpen)}
                >
                    <ChevronDown size={14} />
                </div>
            </div>

            {isOpen && options.length > 0 && (
                <div className="absolute top-full left-0 right-0 mt-1 z-50 glass-panel max-h-60 overflow-y-auto border border-gray-600 shadow-xl bg-slate-900">
                    <div
                        className="flex items-center justify-between px-3 py-2 text-sm hover:bg-violet-600/50 cursor-pointer text-gray-200 transition-colors"
                        onClick={() => {
                            onChange('');
                            setIsOpen(false);
                        }}
                    >
                        <span className="text-gray-400 italic">-- No Voice --</span>
                    </div>

                    {options.map(opt => {
                        const voiceId = opt.path || opt.id;
                        const isPlaying = playingUrl === api.getVoicePreviewUrl(voiceId);
                        const diffName = getCleanName(opt.name);

                        return (
                            <div
                                key={voiceId}
                                className="flex items-center justify-between px-3 py-2 text-sm hover:bg-violet-600/50 cursor-pointer text-gray-200 transition-colors"
                                onClick={() => {
                                    onChange(voiceId);
                                    setIsOpen(false);
                                }}
                            >
                                <div className="truncate flex-1 mr-2" title={voiceId}>
                                    {diffName}
                                </div>
                                <button
                                    className="text-gray-400 hover:text-white p-1 rounded hover:bg-white/10"
                                    onClick={(e) => handlePlay(e, voiceId)}
                                    title={isPlaying ? "Stop" : "Preview"}
                                >
                                    {isPlaying ? <Square size={14} fill="currentColor" /> : <Play size={14} fill="currentColor" />}
                                </button>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
