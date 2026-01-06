import React, { useState, useEffect } from 'react';
import { Users, Merge, Mic, Save, Check, ArrowUpDown, ArrowUp, ArrowDown, ChevronDown } from 'lucide-react';
import VoiceSelect from './VoiceSelect';

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

export default function CharacterManagement({ onBack }) {
    const [characters, setCharacters] = useState([]);
    const [voiceMapping, setVoiceMapping] = useState({});
    const [voices, setVoices] = useState([]);
    const [selectedChars, setSelectedChars] = useState([]);
    const [mergeTarget, setMergeTarget] = useState('');
    const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);
    const [message, setMessage] = useState('');
    const [sortConfig, setSortConfig] = useState({ key: 'count', direction: 'desc' });

    // Voice Modal State


    useEffect(() => {
        fetchCharacters();
        fetchVoices();
    }, []);

    const fetchCharacters = async () => {
        try {
            const res = await fetch('http://localhost:8080/api/characters');
            const data = await res.json();
            setCharacters(data.characters || []);
            setVoiceMapping(data.mapping || {});
        } catch (err) {
            console.error("Failed to fetch characters", err);
        }
    };

    const fetchVoices = async () => {
        try {
            const res = await fetch('http://localhost:8080/api/voices/list');
            const data = await res.json();
            setVoices(data.voices || []);
        } catch (err) {
            console.error("Failed to fetch voices", err);
        }
    };

    const handleSelectChar = (char) => {
        if (selectedChars.includes(char)) {
            setSelectedChars(selectedChars.filter(c => c !== char));
        } else {
            setSelectedChars([...selectedChars, char]);
        }
    };

    const openMergeModal = () => {
        if (selectedChars.length < 2) return;
        setMergeTarget(selectedChars[0]); // Default to first selected
        setIsMergeModalOpen(true);
    };

    const confirmMerge = async () => {
        try {
            const res = await fetch('http://localhost:8080/api/characters/merge', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    target: mergeTarget,
                    // backend logic handles merge, ensuring target is preserved
                    sources: selectedChars.filter(c => c !== mergeTarget)
                })
            });
            const data = await res.json();
            if (data.error) throw new Error(data.error);

            setMessage(`Merged ${data.count} uses successfully.`);
            setIsMergeModalOpen(false);
            setSelectedChars([]);
            fetchCharacters(); // Refresh list

            setTimeout(() => setMessage(''), 3000);
        } catch (err) {
            alert('Merge failed: ' + err.message);
        }
    };

    const updateCharacterConfig = async (char, field, value) => {
        // Optimistic update
        const currentConfig = voiceMapping[char] || {};
        const updatedConfig = { ...currentConfig, [field]: value };

        const newMapping = { ...voiceMapping, [char]: updatedConfig };
        setVoiceMapping(newMapping);

        // API call
        try {
            const res = await fetch('http://localhost:8080/api/characters/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: char,
                    voiceConfig: updatedConfig
                })
            });
            if (!res.ok) throw new Error('Update failed');
        } catch (err) {
            console.error(err);
            // Revert? simpler to just fetch again or ignore
        }
    };

    const handleSort = (key) => {
        let direction = 'asc';
        if (sortConfig.key === key && sortConfig.direction === 'asc') {
            direction = 'desc';
        }
        setSortConfig({ key, direction });
    };

    const sortedCharacters = [...characters].sort((a, b) => {
        if (sortConfig.key === 'count') {
            const countA = a.chapters.length;
            const countB = b.chapters.length;
            if (countA !== countB) {
                return sortConfig.direction === 'asc' ? countA - countB : countB - countA;
            }
            // Tie-break with name
            return a.name.localeCompare(b.name);
        }
        if (sortConfig.key === 'name') {
            return sortConfig.direction === 'asc'
                ? a.name.localeCompare(b.name)
                : b.name.localeCompare(a.name);
        }
        return 0;
    });

    const SortIcon = ({ column }) => {
        if (sortConfig.key !== column) return <ArrowUpDown size={14} className="opacity-30" />;
        return sortConfig.direction === 'asc' ? <ArrowUp size={14} className="text-violet-400" /> : <ArrowDown size={14} className="text-violet-400" />;
    };

    return (
        <div className="h-full flex flex-col p-6 overflow-auto">
            <div className="glass-panel p-6 flex flex-col h-full">
                <div className="flex justify-between items-center mb-6">
                    <div className="flex items-center gap-3">
                        <button onClick={onBack} className="text-gray-400 hover:text-white transition-colors">← Back</button>
                        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
                            <Users className="w-6 h-6" /> Character Management
                        </h1>
                    </div>

                    {message && <div className="bg-green-500/10 text-green-400 border border-green-500/20 px-4 py-2 rounded flex items-center gap-2"><Check size={16} />{message}</div>}

                    <div className="flex gap-3">
                        {selectedChars.length >= 2 && (
                            <button
                                onClick={openMergeModal}
                                className="bg-violet-600 hover:bg-violet-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 shadow-sm transition-colors"
                            >
                                <Merge size={18} /> Merge Selected ({selectedChars.length})
                            </button>
                        )}
                    </div>
                </div>

                <div className="overflow-auto flex-1">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="text-gray-400 border-b border-gray-700/50">
                                <th className="p-4 w-12 text-center">
                                    <input
                                        type="checkbox"
                                        onChange={(e) => {
                                            if (e.target.checked) setSelectedChars(characters.map(c => c.name));
                                            else setSelectedChars([]);
                                        }}
                                        checked={characters.length > 0 && selectedChars.length === characters.length}
                                        className="rounded border-gray-600 bg-gray-700/50 text-violet-500 focus:ring-violet-500 focus:ring-offset-gray-800"
                                    />
                                </th>
                                <th
                                    className="p-4 cursor-pointer hover:text-white transition-colors group select-none"
                                    onClick={() => handleSort('name')}
                                >
                                    <div className="flex items-center gap-2">
                                        Character Name
                                        <SortIcon column="name" />
                                    </div>
                                </th>
                                <th
                                    className="p-4 cursor-pointer hover:text-white transition-colors group select-none"
                                    onClick={() => handleSort('count')}
                                >
                                    <div className="flex items-center gap-2">
                                        Appears In
                                        <SortIcon column="count" />
                                    </div>
                                </th>
                                <th className="p-4 w-1/4">Assigned Voice</th>
                                <th className="p-4 w-32">Default Emotion</th>
                            </tr>
                        </thead>
                        <tbody>
                            {characters.length === 0 && (
                                <tr><td colSpan="5" className="p-8 text-center text-gray-500">No characters found yet. Analyze some chapters!</td></tr>
                            )}
                            {sortedCharacters.map(char => (
                                <tr key={char.name} className="border-b border-gray-700/50 hover:bg-white/5 transition-colors">
                                    <td className="p-4 text-center">
                                        <input
                                            type="checkbox"
                                            checked={selectedChars.includes(char.name)}
                                            onChange={() => handleSelectChar(char.name)}
                                            className="rounded border-gray-600 bg-gray-700/50 text-violet-500 focus:ring-violet-500 focus:ring-offset-gray-800"
                                        />
                                    </td>
                                    <td className="p-4 font-medium text-gray-200">{char.name}</td>
                                    <td className="p-4 text-sm text-gray-400 max-w-xs truncate" title={char.chapters.join(', ')}>
                                        <span className="bg-gray-700/50 px-2 py-0.5 rounded text-xs mr-2 text-gray-300 font-mono">
                                            {char.chapters.length}
                                        </span>
                                        {char.chapters.slice(0, 3).join(', ')}
                                        {char.chapters.length > 3 && ` +${char.chapters.length - 3} more`}
                                    </td>
                                    <td className="p-4 w-64">
                                        <VoiceSelect
                                            value={voiceMapping[char.name]?.voiceId || ''}
                                            onChange={(val) => updateCharacterConfig(char.name, 'voiceId', val)}
                                            options={voices}
                                        />
                                    </td>
                                    <td className="p-4">
                                        <select
                                            className="bg-slate-800 border border-gray-600 text-gray-200 rounded px-2 py-1 text-sm w-full focus:ring-2 focus:ring-violet-500/50 focus:border-violet-500 outline-none"
                                            value={voiceMapping[char.name]?.emotion || 'calm'}
                                            onChange={(e) => updateCharacterConfig(char.name, 'emotion', e.target.value)}
                                        >
                                            {Object.keys(emotionMap).map(em => (
                                                <option key={em} value={em}>{emotionMap[em]}</option>
                                            ))}
                                        </select>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                {/* Merge Modal */}
                {isMergeModalOpen && (
                    <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
                        <div className="glass-panel p-6 w-[400px] shadow-2xl">
                            <h3 className="text-xl font-bold mb-4 flex items-center gap-2 text-white">
                                <Merge className="text-violet-500" /> Merge Characters
                            </h3>
                            <p className="text-gray-400 mb-4 text-sm">
                                Merging <strong>{selectedChars.length}</strong> characters. Select the primary name to keep. All other names will be replaced in the analysis.
                            </p>

                            <div className="mb-6">
                                <label className="block text-sm font-medium text-gray-300 mb-2">Target Name</label>
                                <select
                                    className="w-full bg-slate-800 border border-gray-600 text-white p-2 rounded-lg focus:ring-2 focus:ring-violet-500 focus:border-violet-500 outline-none"
                                    value={mergeTarget}
                                    onChange={(e) => setMergeTarget(e.target.value)}
                                >
                                    {selectedChars.map(c => (
                                        <option key={c} value={c}>{c}</option>
                                    ))}
                                </select>
                                <div className="mt-2 text-xs text-gray-500">
                                    Or type a new name:
                                </div>
                                <input
                                    type="text"
                                    placeholder="Custom target name..."
                                    className="w-full bg-slate-800 border border-gray-600 text-white p-2 rounded-lg mt-1 focus:ring-2 focus:ring-violet-500 focus:border-violet-500 outline-none"
                                    onChange={(e) => setMergeTarget(e.target.value)}
                                />
                            </div>

                            <div className="flex justify-end gap-3">
                                <button
                                    onClick={() => setIsMergeModalOpen(false)}
                                    className="px-4 py-2 text-gray-400 hover:bg-gray-800 rounded-lg transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={confirmMerge}
                                    className="px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700"
                                >
                                    Confirm Merge
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
