import React, { useState, useEffect, useRef } from 'react';
import { api } from '../services/api';
import { WebSocketClient } from '../services/ws';
import { Play, Pause, Loader, CheckCircle, RefreshCw } from 'lucide-react';

export default function Player({ chapterId }) {
    const [progress, setProgress] = useState(0);
    const [logs, setLogs] = useState([]);
    const [isPlaying, setIsPlaying] = useState(false); // UI state for generation active
    const [audioUrl, setAudioUrl] = useState(null);
    const [hasStarted, setHasStarted] = useState(false);

    useEffect(() => {
        // Connect WS
        const ws = new WebSocketClient('ws://localhost:8080/api/ws', (msg) => {
            if (msg.type === 'progress' && msg.chapterId === chapterId) {
                setProgress(msg.percentage);
                setLogs(prev => [msg.message, ...prev].slice(0, 5)); // Keep last 5 logs
                if (msg.percentage === 100) {
                    // Update URL by checking status again
                    api.checkAudioStatus(chapterId).then(res => {
                        if (res.data.exists && res.data.url) {
                            setAudioUrl(res.data.url);
                        }
                    });
                }
            }
        });
        ws.connect();

        return () => {
            ws.close();
        };
    }, [chapterId]);

    // Check if audio already exists when component mounts
    useEffect(() => {
        const checkExistingAudio = async () => {
            try {
                const response = await api.checkAudioStatus(chapterId);
                if (response.data.exists) {
                    setHasStarted(true);
                    setProgress(100);
                    setAudioUrl(response.data.url);
                    setLogs(['音频已生成']);
                }
            } catch (err) {
                console.error('Failed to check audio status:', err);
            }
        };
        checkExistingAudio();
    }, [chapterId]);

    const handleStart = async () => {
        setHasStarted(true);
        try {
            await api.generateAudio(chapterId);
        } catch (err) {
            setLogs(p => [`启动失败: ${err}`, ...p]);
            setHasStarted(false); // Reset on immediate failure
        }
    };

    if (!hasStarted) {
        return (
            <div className="glass-panel p-8 text-center flex flex-col items-center justify-center min-h-[300px]">
                <h3 className="text-2xl font-bold mb-4">准备生成</h3>
                <p className="text-gray-400 mb-8 max-w-md">
                    语音映射已确认。点击下方按钮开始生成音频。
                    根据章节长度，此过程可能需要一段时间。
                </p>
                <button
                    onClick={handleStart}
                    className="btn-primary px-8 py-4 text-lg flex items-center gap-3"
                >
                    <Play size={24} fill="currentColor" /> 开始生成
                </button>
            </div>
        );
    }

    return (
        <div className="glass-panel p-8 text-center">
            <h3 className="text-2xl font-bold mb-6">正在生成音频...</h3>

            <div className="relative w-full h-4 bg-gray-700 rounded-full overflow-hidden mb-4">
                <div
                    className="absolute top-0 left-0 h-full bg-violet-500 transition-all duration-300 ease-out"
                    style={{ width: `${progress}%` }}
                />
            </div>
            <div className="text-violet-300 font-mono text-xl mb-8">{progress}%</div>

            <div className="bg-black/30 rounded-lg p-4 text-left h-32 overflow-hidden text-sm font-mono text-gray-400">
                {logs.map((log, i) => (
                    <div key={i}>{log}</div>
                ))}
            </div>

            {progress === 100 && (
                <div className="mt-8">
                    <div className="flex items-center justify-center gap-4 mb-4">
                        <h4 className="text-green-400 flex items-center gap-2">
                            <CheckCircle /> 生成完成
                        </h4>
                        <button
                            onClick={handleStart}
                            className="text-xs flex items-center gap-1 bg-white/10 hover:bg-white/20 px-3 py-1 rounded transition-colors"
                            title="重新生成音频"
                        >
                            <RefreshCw size={14} /> 重新生成
                        </button>
                    </div>

                    {/* Audio Element */}
                    {audioUrl && (
                        <audio controls className="w-full" key={audioUrl}>
                            <source src={`http://localhost:8080${audioUrl}`} type="audio/wav" />
                            您的浏览器不支持音频播放。
                        </audio>
                    )}
                </div>
            )}
        </div>
    );
}


