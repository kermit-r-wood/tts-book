import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { Settings as SettingsIcon, Save, RefreshCw } from 'lucide-react';

export default function Settings() {
    const [config, setConfig] = useState({
        llm_api_key: '',
        llm_base_url: '',
        llm_model: '',
        llm_provider: 'openai',
        index_tts_url: '',
        llm_chunk_size: 1000,
        llm_min_interval: 3000,
        merge_silence: 400,
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
                setStatus('模型获取成功');
                setTimeout(() => setStatus(''), 2000);
            }
        } catch (e) {
            console.error(e);
            setStatus('模型获取失败');
        } finally {
            setLoadingModels(false);
        }
    };

    const handleSave = async () => {
        try {
            await api.updateConfig(config);
            setStatus('保存成功！');
            setTimeout(() => setStatus(''), 2000);
        } catch (e) {
            setStatus('保存设置失败');
        }
    };

    return (
        <div className="glass-panel p-6">
            <div className="flex items-center gap-2 mb-6 text-xl font-bold text-violet-400">
                <SettingsIcon /> 设置
            </div>

            <div className="space-y-4">
                <div>
                    <label className="block text-sm text-gray-400 mb-1">Index-TTS API 地址</label>
                    <input
                        className="input-field"
                        value={config.index_tts_url}
                        onChange={e => setConfig({ ...config, index_tts_url: e.target.value })}
                        placeholder="http://127.0.0.1:7860"
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">语音文件夹 (可选)</label>
                    <input
                        className="input-field"
                        value={config.voice_dir || ''}
                        onChange={e => setConfig({ ...config, voice_dir: e.target.value })}
                        placeholder="存放语音文件的文件夹路径 (例如 C:\Voices)"
                    />
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">API Provider</label>
                    <select
                        className="input-field appearance-none"
                        value={config.llm_provider || 'openai'}
                        onChange={e => setConfig({ ...config, llm_provider: e.target.value })}
                    >
                        <option value="openai">OpenAI Compatible (ChatGPT, DeepSeek, etc.)</option>
                        <option value="gemini">Google Gemini (Native)</option>
                    </select>
                </div>

                <div>
                    <label className="block text-sm text-gray-400 mb-1">大模型 API Key</label>
                    <input
                        className="input-field"
                        type="password"
                        value={config.llm_api_key || ''}
                        onChange={e => setConfig({ ...config, llm_api_key: e.target.value })}
                        placeholder="sk-..."
                    />
                </div>



                {config.llm_provider === 'openai' && (
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">大模型 Base URL</label>
                        <input
                            className="input-field"
                            value={config.llm_base_url || ''}
                            onChange={e => setConfig({ ...config, llm_base_url: e.target.value })}
                            placeholder="https://api..."
                        />
                    </div>
                )}

                <div>
                    <label className="block text-sm text-gray-400 mb-1">大模型</label>
                    <div className="flex gap-2">
                        <div className="relative flex-1">
                            {availableModels.length > 0 ? (
                                <select
                                    className="input-field appearance-none"
                                    value={config.llm_model || ''}
                                    onChange={e => setConfig({ ...config, llm_model: e.target.value })}
                                >
                                    <option value="">选择模型...</option>
                                    {availableModels.map(m => (
                                        <option key={m} value={m}>{m}</option>
                                    ))}
                                </select>
                            ) : (
                                <input
                                    className="input-field"
                                    value={config.llm_model || ''}
                                    onChange={e => setConfig({ ...config, llm_model: e.target.value })}
                                    placeholder="手动输入模型名称 (或点击刷新)"
                                />
                            )}
                        </div>
                        <button
                            className="btn-secondary px-3"
                            onClick={fetchModels}
                            title="刷新可用模型"
                            disabled={loadingModels}
                        >
                            <RefreshCw size={18} className={loadingModels ? "animate-spin" : ""} />
                        </button>
                    </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">最大分块长度 (字符)</label>
                        <input
                            type="number"
                            className="input-field"
                            value={config.llm_chunk_size}
                            onChange={e => setConfig({ ...config, llm_chunk_size: parseInt(e.target.value) || 0 })}
                            placeholder="1000"
                        />
                    </div>
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">最小请求间隔 (ms)</label>
                        <input
                            type="number"
                            className="input-field"
                            value={config.llm_min_interval}
                            onChange={e => setConfig({ ...config, llm_min_interval: parseInt(e.target.value) || 0 })}
                            placeholder="3000"
                        />
                    </div>
                    <div>
                        <label className="block text-sm text-gray-400 mb-1">段落间隔静音 (ms)</label>
                        <input
                            type="number"
                            className="input-field"
                            value={config.merge_silence}
                            onChange={e => setConfig({ ...config, merge_silence: parseInt(e.target.value) || 0 })}
                            placeholder="400"
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
                        启用模拟大模型 (模拟响应，无 API 消耗)
                    </label>
                </div>

                <button className="btn-primary w-full justify-center mt-4" onClick={handleSave}>
                    <Save size={18} /> 保存设置
                </button>

                {status && <div className="text-center text-sm text-green-400 mt-2">{status}</div>}
            </div>
        </div >
    );
}
