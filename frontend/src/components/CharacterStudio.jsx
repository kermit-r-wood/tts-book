import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { User, CheckCircle } from 'lucide-react';
import CharacterRow from './CharacterRow';

export default function CharacterStudio({ chapterId, onGenerate, embedded, analysisData }) {
    const [characters, setCharacters] = useState([]);
    const [mapping, setMapping] = useState({});
    const [loading, setLoading] = useState(true);
    const [confirming, setConfirming] = useState(false);

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
            const localChars = [...new Set(analysis.map(s => s.speaker).filter(Boolean))];

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
            const response = await api.confirmMapping(mapping);
            console.log('Mapping confirmed:', response.data);
            onGenerate(chapterId);
        } catch (error) {
            console.error('Failed to confirm mapping:', error);
            if (error.response) {
                // Server responded with error status
                console.error('Response data:', error.response.data);
                console.error('Response status:', error.response.status);
                alert(`Failed to save voice mapping: ${error.response.data?.error || error.response.statusText}`);
            } else if (error.request) {
                // Request was made but no response received
                console.error('No response received:', error.request);
                alert('Failed to save voice mapping: No response from server. Please check if the backend is running.');
            } else {
                // Something else happened
                console.error('Error message:', error.message);
                alert(`Failed to save voice mapping: ${error.message}`);
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

    if (loading) return <div className="p-8 text-center text-gray-400">Loading Characters...</div>;

    const containerClass = embedded ? '' : 'glass-panel p-6';

    return (
        <div className={containerClass}>
            <div className="flex justify-between items-center mb-6">
                <h3 className="text-xl font-bold flex items-center gap-2">
                    <User /> Character Voice Map
                </h3>
                <button
                    className="btn-primary"
                    onClick={confirm}
                    disabled={confirming}
                >
                    <CheckCircle size={18} /> {confirming ? 'Confirming...' : 'Confirm & Next'}
                </button>
            </div>

            <div className="overflow-x-auto min-h-[300px]">
                <table className="w-full text-left border-collapse">
                    <thead>
                        <tr className="text-gray-400 border-b border-gray-700">
                            <th className="p-3">Character</th>
                            <th className="p-3 w-1/3">Voice Path / URL</th>
                            <th className="p-3">Emotion (Default)</th>
                            <th className="p-3">Speed</th>
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
                            />
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
