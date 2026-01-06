import React, { useState, useRef, useEffect } from 'react';
import { X, Play, Square, Search, Check } from 'lucide-react';
import { api } from '../services/api';

export default function VoiceSelectionModal({ isOpen, onClose, onSelect, voices, currentVoiceId, charName }) {
    const [searchTerm, setSearchTerm] = useState('');
    const [playingUrl, setPlayingUrl] = useState(null);
    const audioRef = useRef(null);

    // Filter voices
    const filteredVoices = voices.filter(v =>
        v.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        v.id.toLowerCase().includes(searchTerm.toLowerCase())
    );

    // Cleanup audio on unmount or close
    useEffect(() => {
        if (!isOpen) {
            stopAudio();
        }
        return () => stopAudio();
    }, [isOpen]);

    const stopAudio = () => {
        if (audioRef.current) {
            audioRef.current.pause();
            audioRef.current = null;
        }
        setPlayingUrl(null);
    };

    const handlePlay = (e, path) => {
        e.stopPropagation();

        const url = api.getVoicePreviewUrl(path);

        if (playingUrl === url) {
            stopAudio();
            return;
        }

        stopAudio();

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

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="glass-panel w-full max-w-2xl h-[80vh] flex flex-col shadow-2xl border border-gray-600 animate-in fade-in zoom-in duration-200">
                {/* Header */}
                <div className="p-4 border-b border-gray-700 flex justify-between items-center bg-slate-900/50">
                    <h2 className="text-xl font-bold text-white">
                        Select Voice for <span className="text-violet-400">{charName}</span>
                    </h2>
                    <button onClick={onClose} className="text-gray-400 hover:text-white p-1 rounded-full hover:bg-white/10 transition-colors">
                        <X size={24} />
                    </button>
                </div>

                {/* Search */}
                <div className="p-4 border-b border-gray-700 bg-slate-900/30">
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
                        <input
                            type="text"
                            placeholder="Search voices..."
                            className="w-full bg-slate-800 border border-gray-600 text-white pl-10 pr-4 py-2 rounded-lg focus:ring-2 focus:ring-violet-500 outline-none"
                            value={searchTerm}
                            onChange={e => setSearchTerm(e.target.value)}
                            autoFocus
                        />
                    </div>
                </div>

                {/* List */}
                <div className="flex-1 overflow-y-auto p-2 space-y-1">
                    <div
                        className={`p-3 rounded-lg flex items-center justify-between cursor-pointer transition-colors ${currentVoiceId === '' ? 'bg-violet-600/20 border border-violet-500/50' : 'hover:bg-white/5 border border-transparent'
                            }`}
                        onClick={() => onSelect('')}
                    >
                        <span className="text-gray-300 font-medium">-- No Voice --</span>
                        {currentVoiceId === '' && <Check size={18} className="text-violet-400" />}
                    </div>

                    {filteredVoices.map(voice => {
                        const isSelected = currentVoiceId === voice.id;
                        const isPlaying = playingUrl === api.getVoicePreviewUrl(voice.id); // voice.id is the path usually

                        return (
                            <div
                                key={voice.id}
                                className={`p-3 rounded-lg flex items-center justify-between group transition-colors ${isSelected ? 'bg-violet-600/20 border border-violet-500/50' : 'hover:bg-white/5 border border-transparent'
                                    }`}
                                onClick={() => onSelect(voice.id)}
                            >
                                <div className="flex flex-col cursor-pointer flex-1">
                                    <span className={`font-medium ${isSelected ? 'text-violet-300' : 'text-gray-200'}`}>
                                        {voice.name}
                                    </span>
                                    {voice.language && (
                                        <span className="text-xs text-gray-500">{voice.language}</span>
                                    )}
                                </div>

                                <div className="flex items-center gap-3">
                                    <button
                                        className={`p-2 rounded-full hover:bg-violet-600 text-gray-400 hover:text-white transition-colors ${isPlaying ? 'bg-violet-600 text-white' : ''}`}
                                        onClick={(e) => handlePlay(e, voice.id)}
                                        title="Preview Voice"
                                    >
                                        {isPlaying ? <Square size={16} fill="currentColor" /> : <Play size={16} fill="currentColor" />}
                                    </button>

                                    {isSelected && <Check size={18} className="text-violet-400" />}
                                </div>
                            </div>
                        );
                    })}

                    {filteredVoices.length === 0 && (
                        <div className="text-center text-gray-500 p-8">
                            No voices found matching "{searchTerm}"
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-gray-700 bg-slate-900/50 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-gray-300 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
}
