import React, { useState, useEffect, useRef } from 'react';
import { api } from '../services/api';
import { WebSocketClient } from '../services/ws';
import { Play, Pause, Loader } from 'lucide-react';

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
                    // Assume output file is accessible. 
                    // For MVP just standard name or return URL in msg.
                }
            }
        });
        ws.connect();

        return () => {
            ws.close();
        };
    }, [chapterId]);

    const handleStart = async () => {
        setHasStarted(true);
        try {
            await api.generateAudio(chapterId);
        } catch (err) {
            setLogs(p => [`Error starting: ${err}`, ...p]);
            setHasStarted(false); // Reset on immediate failure
        }
    };

    if (!hasStarted) {
        return (
            <div className="glass-panel p-8 text-center flex flex-col items-center justify-center min-h-[300px]">
                <h3 className="text-2xl font-bold mb-4">Ready to Generate</h3>
                <p className="text-gray-400 mb-8 max-w-md">
                    Voice mappings are confirmed. Click below to start the audio generation process.
                    This may take a while depending on the chapter length.
                </p>
                <button
                    onClick={handleStart}
                    className="btn-primary px-8 py-4 text-lg flex items-center gap-3"
                >
                    <Play size={24} fill="currentColor" /> Start Generation
                </button>
            </div>
        );
    }

    return (
        <div className="glass-panel p-8 text-center">
            <h3 className="text-2xl font-bold mb-6">Generating Audio...</h3>

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
                    <h4 className="text-green-400 mb-4 flex items-center justify-center gap-2">
                        <CheckCircle /> Generation Complete
                    </h4>
                    {/* Audio Element */}
                    <audio controls className="w-full">
                        <source src={`http://localhost:8080/output/${chapterId}.wav`} type="audio/wav" />
                        Your browser does not support the audio element.
                    </audio>
                </div>
            )}
        </div>
    );
}

import { CheckCircle } from 'lucide-react';
