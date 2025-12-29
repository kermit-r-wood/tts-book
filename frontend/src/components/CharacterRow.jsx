import React, { useState, useRef, useEffect } from 'react';
import { Folder, Globe, ChevronDown } from 'lucide-react';

export default function CharacterRow({ char, mappingData, updateMapping, voiceOptions, handleLocalFileSelect }) {
    const [isOpen, setIsOpen] = useState(false);
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

    return (
        <tr className="border-b border-gray-700/50 hover:bg-white/5">
            <td className="p-3 font-medium text-violet-300">{char}</td>
            <td className="p-3">
                <div className="flex gap-2 items-center">
                    <div className="flex-1 relative" ref={dropdownRef}>
                        <div className="relative">
                            <input
                                className="input-field text-sm py-1 w-full pl-2 pr-8"
                                placeholder="Path or Select ->"
                                value={mappingData?.voiceId || ''}
                                onChange={e => updateMapping(char, 'voiceId', e.target.value)}
                                onClick={() => setIsOpen(!isOpen)}
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
                                {voiceOptions.map(opt => (
                                    <div
                                        key={opt.path}
                                        className="px-3 py-2 text-sm hover:bg-violet-600/50 cursor-pointer text-gray-200 transition-colors truncate"
                                        title={opt.path}
                                        onClick={() => {
                                            updateMapping(char, 'voiceId', opt.path);
                                            setIsOpen(false);
                                        }}
                                    >
                                        {opt.name}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    <div className="flex gap-1">
                        <label className="btn-secondary px-2 py-1 text-xs flex items-center gap-1 cursor-pointer" title="Find Local File">
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
                    className="input-field text-sm py-1 bg-slate-800"
                    value={mappingData?.emotion || 'calm'}
                    onChange={e => updateMapping(char, 'emotion', e.target.value)}
                >
                    {['happy', 'angry', 'sad', 'afraid', 'disgusted', 'melancholic', 'surprised', 'calm'].map(em => (
                        <option key={em} value={em}>{em}</option>
                    ))}
                </select>
            </td>
            <td className="p-3">
                <input
                    type="number" step="0.1"
                    className="input-field text-sm py-1 w-20"
                    placeholder="1.0"
                    value={mappingData?.speed || 1.0}
                    onChange={e => updateMapping(char, 'speed', parseFloat(e.target.value))}
                />
            </td>
        </tr>
    );
}
