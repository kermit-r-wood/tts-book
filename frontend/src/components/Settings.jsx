import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { Settings as SettingsIcon, Save, RefreshCw } from 'lucide-react';

export default function Settings() {
    const [config, setConfig] = useState({
        llm_api_key: '',
        llm_base_url: '',
        llm_model: '',
        index_tts_url: '',
        llm_chunk_size: 1000,
        llm_min_interval: 3000,
        mock_llm: false,
        voice_dir: ''
    });
    const [status, setStatus] = useState('');
    const [availableModels, setAvailableModels] = useState([]);
    const [loadingModels, setLoadingModels] = useState(false);

    useEffect(() => {
        api.getConfig().then(res => setConfig(res.data)).catch(console.error);
    }, []);

    const fetchModels = async () => {
        setLoadingModels(true);
        try {
            const res = await api.getLLMModels();
            if (res.data.models) {
                setAvailableModels(res.data.models);
                setStatus('Models fetched successfully');
                setTimeout(() => setStatus(''), 2000);
            }
        } catch (e) {
            console.error(e);
            setStatus('Failed to fetch models');
        } finally {
            setLoadingModels(false);
        }
    };

    const handleSave = async () => {
        try {
            await api.updateConfig(config);
            setStatus('Saved successfully!');
            setTimeout(() => setStatus(''), 2000);
        } catch (e) {
            setStatus('Error saving settings');
        }
    };

    return (
        <div className="glass-panel p-6">
            <div className="flex items-center gap-2 mb-6 text-xl font-bold text-violet-400">
                <SettingsIcon /> Settings
            </div>

            <div className="space-y-4">
                <div>
                    <label className="block text-sm text-gray-400 mb-1">Index-TTS API URL</label>
                    <input
                        className="input-field"
                        value={config.index_tts_url}
                        onChange={e => setConfig({ ...config, index_tts_url: e.target.value })}
                        placeholder="http://127.0.0.1:7860"
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">Voice Directory (Optional)</label>
                    <input
                        className="input-field"
                        value={config.voice_dir || ''}
                        onChange={e => setConfig({ ...config, voice_dir: e.target.value })}
                        placeholder="Path to folder with voice files (e.g. C:\Voices)"
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">LLM API Key</label>
                    <input
                        className="input-field"
                        type="password"
                        value={config.llm_api_key || ''}
                        onChange={e => setConfig({ ...config, llm_api_key: e.target.value })}
                        placeholder="sk-..."
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">LLM Base URL</label>
                    <input
                        className="input-field"
                        value={config.llm_base_url || ''}
                        onChange={e => setConfig({ ...config, llm_base_url: e.target.value })}
                        placeholder="https://api..."
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">LLM Model</label>
                    <div className="flex gap-2">
                        <div className="relative flex-1">
                            {availableModels.length > 0 ? (
                                <select
                                    className="input-field appearance-none"
                                    value={config.llm_model || ''}
                                    onChange={e => setConfig({ ...config, llm_model: e.target.value })}
                                >
                                    <option value="">Select a model...</option>
                                    {availableModels.map(m => (
                                        <option key={m} value={m}>{m}</option>
                                    ))}
                                </select>
                            ) : (
                                <input
                                    className="input-field"
                                    value={config.llm_model || ''}
                                    onChange={e => setConfig({ ...config, llm_model: e.target.value })}
                                    placeholder="Enter model name manually (or click fetch)"
                                />
                            )}
                        </div>
                        <button
                            className="btn-secondary px-3"
                            onClick={fetchModels}
                            title="Fetch available models"
                            disabled={loadingModels}
                        >
                            <RefreshCw size={18} className={loadingModels ? "animate-spin" : ""} />
                        </button>
                    </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Max Chunk Size (chars)</label>
                        <input
                            type="number"
                            className="input-field"
                            value={config.llm_chunk_size}
                            onChange={e => setConfig({ ...config, llm_chunk_size: parseInt(e.target.value) || 0 })}
                            placeholder="1000"
                        />
                    </div>
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">Min Request Interval (ms)</label>
                        <input
                            type="number"
                            className="input-field"
                            value={config.llm_min_interval}
                            onChange={e => setConfig({ ...config, llm_min_interval: parseInt(e.target.value) || 0 })}
                            placeholder="3000"
                        />
                    </div>
                </div>

                <div className="flex items-center gap-2">
                    <input
                        type="checkbox"
                        id="mock_llm"
                        checked={config.mock_llm || false}
                        onChange={e => setConfig({ ...config, mock_llm: e.target.checked })}
                        className="w-4 h-4 rounded border-gray-600 bg-slate-700 text-violet-500 focus:ring-violet-500"
                    />
                    <label htmlFor="mock_llm" className="text-sm text-gray-300 cursor-pointer select-none">
                        Enable Mock LLM (Simulate responses, no API cost)
                    </label>
                </div>

                <button className="btn-primary w-full justify-center mt-4" onClick={handleSave}>
                    <Save size={18} /> Save Settings
                </button>

                {status && <div className="text-center text-sm text-green-400 mt-2">{status}</div>}
            </div>
        </div>
    );
}
