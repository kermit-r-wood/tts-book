import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { User, CheckCircle, Merge, Check } from 'lucide-react';
import CharacterRow from './CharacterRow';

export default function CharacterStudio({ chapterId, onGenerate, embedded, analysisData, onRefresh }) {
    const [characters, setCharacters] = useState([]);
    const [charCounts, setCharCounts] = useState({}); // Store occurrence counts
    const [mapping, setMapping] = useState({});
    const [loading, setLoading] = useState(true);
    const [confirming, setConfirming] = useState(false);

    // Selection & Merging
    const [selectedChars, setSelectedChars] = useState([]);
    const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);
    const [mergeTarget, setMergeTarget] = useState('');
    const [mergeMessage, setMergeMessage] = useState('');

    const [voiceOptions, setVoiceOptions] = useState([]);

    useEffect(() => {
        loadData();
        fetchVoiceOptions();
    }, [chapterId, analysisData]);

    const fetchVoiceOptions = async () => {
        try {
            const res = await api.getVoiceList();
            if (res.data?.voices) {
                setVoiceOptions(res.data.voices);
            }
        } catch (e) {
            console.error("Failed to fetch voice options", e);
        }
    };

    const loadData = async () => {
        try {
            setLoading(true);

            // 1. Get characters from Local Analysis Data (Robust source)
            const analysis = analysisData[chapterId] || [];

            // Calculate occurrences
            const counts = {};
            analysis.forEach(s => {
                if (s.speaker) {
                    counts[s.speaker] = (counts[s.speaker] || 0) + 1;
                }
            });
            setCharCounts(counts);

            // Get unique characters and sort by count (descending)
            const localChars = Object.keys(counts).sort((a, b) => counts[b] - counts[a]);

            // 2. Get existing mapping from Backend
            const res = await api.getCharacters();

            // 3. Merge: Use local chars as base, but respect backend if it has more (rare)
            // Actually, just use localChars for the list, and use mapping for the configs.
            setCharacters(localChars);
            setMapping(res.data.mapping || {});
        } catch (e) {
            console.error(e);
        } finally {
            setLoading(false);
        }
    };

    const updateMapping = (char, field, value) => {
        console.log(`Update ${char} ${field} to ${value}`);
        setMapping(prev => {
            const current = prev[char] || {};
            const updatedChar = {
                ...current,
                [field]: value
            };
            // If we are updating voiceId, we usually want it to be our reference audio too
            if (field === 'voiceId') {
                updatedChar.refAudio = value;
            }
            return {
                ...prev,
                [char]: updatedChar
            };
        });
    };

    const confirm = async () => {
        setConfirming(true);
        try {
            console.log('Sending mapping:', mapping);
            // We only need to save for the characters present in this chapter effectively, 
            // but the API saves whatever we send. Sending all known mapping is safe or just the subset.
            // Let's send all mapping we have in state to be safe.
            const response = await api.confirmMapping(mapping);
            console.log('Mapping confirmed:', response.data);
            onGenerate(chapterId);
        } catch (error) {
            console.error('Failed to confirm mapping:', error);
            if (error.response) {
                // Server responded with error status
                console.error('Response data:', error.response.data);
                console.error('Response status:', error.response.status);
                alert(`保存语音映射失败: ${error.response.data?.error || error.response.statusText}`);
            } else if (error.request) {
                // Request was made but no response received
                console.error('No response received:', error.request);
                alert('保存语音映射失败: 无法连接服务器，请检查后端是否运行。');
            } else {
                // Something else happened
                console.error('Error message:', error.message);
                alert(`保存语音映射失败: ${error.message}`);
            }
        } finally {
            setConfirming(false);
        }
    };

    const handleLocalFileSelect = (char, e) => {
        const file = e.target.files[0];
        if (file) {
            updateMapping(char, 'voiceId', file.name);
        }
    };

    // --- Merge Logic ---

    const handleSelectChar = (char, isSelected) => {
        if (isSelected) {
            setSelectedChars(prev => [...prev, char]);
        } else {
            setSelectedChars(prev => prev.filter(c => c !== char));
        }
    };

    const handleSelectAll = (isChecked) => {
        if (isChecked) {
            setSelectedChars(characters);
        } else {
            setSelectedChars([]);
        }
    };

    const openMergeModal = () => {
        if (selectedChars.length < 2) return;
        setMergeTarget(selectedChars[0]);
        setIsMergeModalOpen(true);
    };

    const confirmMerge = async () => {
        try {
            await fetch('http://localhost:8080/api/characters/merge', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    target: mergeTarget,
                    sources: selectedChars.filter(c => c !== mergeTarget)
                })
            });

            setMergeMessage(`Merged successfully.`);
            setIsMergeModalOpen(false);
            setSelectedChars([]);

            // Reload data to reflect changes (analysis data needs refresh ideally)
            // But since analysisData is passed from parent, we might need a way to refresh parent.
            // For now, reloadData will fetch updated characters list if backend returns it, 
            // BUT analysisData in App.jsx is the source of truth for the list.
            // We need to trigger a refresh in App.jsx. 
            // CharacterStudio doesn't have a callback to refresh App's analysisData.
            // Ideally, we should add one. But for now, we can try reloadData, which fetches 'getCharacters'.
            // However, 'getCharacters' returns ALL characters from DB, whereas `characters` state here is derived from `analysisData`.

            // CRITICAL: We need to update the parent's analysisData because that's what drives this view.
            // The merge API updates the backend analysis. Ideally we should re-fetch the analysis for this chapter.
            // Or force a reload.
            // Let's reload the page or ask user to refresh? No, that's bad UX.
            // We can just rely on `characters` state being updated? 
            // In `loadData`, we use `analysisData`. If `analysisData` is stale, `characters` will be stale.

            // Trigger parent refresh to get updated analysis data (merged names)
            if (onRefresh) {
                onRefresh();
            } else {
                // Fallback: Manually update local state if no refresh callback
                const sources = selectedChars.filter(c => c !== mergeTarget);
                setCharacters(prev => prev.filter(c => !sources.includes(c)));
            }

            setTimeout(() => setMergeMessage(''), 3000);

        } catch (err) {
            alert('Merge failed: ' + err.message);
        }
    };

    if (loading) return <div className="p-8 text-center text-gray-400">正在加载角色...</div>;

    const containerClass = embedded ? '' : 'glass-panel p-6';

    return (
        <div className={containerClass}>
            <div className="flex justify-between items-center mb-6">
                <div className="flex items-center gap-4">
                    <h3 className="text-xl font-bold flex items-center gap-2">
                        <User /> 角色语音映射
                    </h3>
                    {selectedChars.length >= 2 && (
                        <button
                            onClick={openMergeModal}
                            className="bg-violet-600 hover:bg-violet-700 text-white px-3 py-1.5 rounded-lg flex items-center gap-2 text-sm shadow-sm transition-colors"
                        >
                            <Merge size={16} /> 合并选中 ({selectedChars.length})
                        </button>
                    )}
                    {mergeMessage && <div className="text-green-400 text-sm flex items-center gap-1"><Check size={14} />{mergeMessage}</div>}
                </div>
                <button
                    className="btn-primary"
                    onClick={confirm}
                    disabled={confirming}
                >
                    <CheckCircle size={18} /> {confirming ? '确认中...' : '确认并继续'}
                </button>
            </div>

            <div className="overflow-x-auto min-h-[300px]">
                <table className="w-full text-left border-collapse">
                    <thead>
                        <tr className="text-gray-400 border-b border-gray-700">
                            <th className="p-3 w-12">
                                <input
                                    type="checkbox"
                                    className="rounded border-gray-600 bg-gray-700 text-violet-500 focus:ring-violet-500"
                                    checked={characters.length > 0 && selectedChars.length === characters.length}
                                    onChange={(e) => handleSelectAll(e.target.checked)}
                                />
                            </th>
                            <th className="p-3">角色</th>
                            <th className="p-3 w-1/3">语音路径 / 链接</th>
                            <th className="p-3 w-32">默认情感</th>

                            <th className="p-3 w-32">情感模式</th>
                        </tr>
                    </thead>
                    <tbody>
                        {characters.map(char => (
                            <CharacterRow
                                key={char}
                                char={char}
                                mappingData={mapping[char]}
                                updateMapping={updateMapping}
                                voiceOptions={voiceOptions}
                                handleLocalFileSelect={handleLocalFileSelect}
                                occurrenceCount={charCounts[char]}
                                selected={selectedChars.includes(char)}
                                onSelect={handleSelectChar}
                            />
                        ))}
                    </tbody>
                </table>
            </div>

            {/* Merge Modal - Reused Style */}
            {isMergeModalOpen && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
                    <div className="glass-panel p-6 w-[400px] shadow-2xl border border-gray-600">
                        <h3 className="text-xl font-bold mb-4 flex items-center gap-2 text-white">
                            <Merge className="text-violet-500" /> 合并角色
                        </h3>
                        <p className="text-gray-400 mb-4 text-sm">
                            将 <strong>{selectedChars.length}</strong> 个角色合并。请选择保留的角色名。其他名字将被替换。
                        </p>

                        <div className="mb-6">
                            <label className="block text-sm font-medium text-gray-300 mb-2">保留名称 (Target)</label>
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
                                或者输入新名称:
                            </div>
                            <input
                                type="text"
                                placeholder="输入新名称..."
                                className="w-full bg-slate-800 border border-gray-600 text-white p-2 rounded-lg mt-1 focus:ring-2 focus:ring-violet-500 focus:border-violet-500 outline-none"
                                onChange={(e) => setMergeTarget(e.target.value)}
                            />
                        </div>

                        <div className="flex justify-end gap-3">
                            <button
                                onClick={() => setIsMergeModalOpen(false)}
                                className="px-4 py-2 text-gray-400 hover:bg-gray-800 rounded-lg transition-colors"
                            >
                                取消
                            </button>
                            <button
                                onClick={confirmMerge}
                                className="px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700"
                            >
                                确认合并
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
