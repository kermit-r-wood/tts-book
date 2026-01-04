import React, { useState, useEffect, useRef } from 'react';
import { api } from '../services/api';
import { Upload, FileText, ChevronRight, Sparkles } from 'lucide-react';

export default function ChapterList({ chapters, setChapters, onSelectChapter, batchProgress, setBatchProgress }) {
    const [uploading, setUploading] = useState(false);
    const wsRef = useRef(null);

    const handleUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        setUploading(true);
        try {
            const res = await api.uploadEpub(file);
            setChapters(res.data.chapters);
        } catch (err) {
            console.error(err);
            alert('Upload failed');
        } finally {
            setUploading(false);
        }
    };

    // Setup WebSocket for progress updates
    useEffect(() => {
        const ws = new WebSocket('ws://localhost:8080/api/ws');
        wsRef.current = ws;

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'progress' && data.chapterId === 'batch') {
                setBatchProgress({
                    percent: data.percentage,
                    message: data.message,
                    analyzing: data.percentage < 100
                });
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        return () => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.close();
            }
        };
    }, []);

    const handleAnalyzeAll = async () => {
        if (chapters.length === 0) {
            alert('No chapters to analyze');
            return;
        }

        setBatchProgress({ percent: 0, message: 'Starting batch analysis...', analyzing: true });

        try {
            await api.analyzeAllChapters(false);
        } catch (err) {
            console.error(err);
            alert('Batch analysis failed to start');
            setBatchProgress({ percent: 0, message: '', analyzing: false });
        }
    };

    return (
        <div className="space-y-6">
            <div className="glass-panel p-8 text-center border-dashed border-2 border-gray-600 hover:border-violet-500 transition-colors">
                <Upload className="mx-auto h-12 w-12 text-gray-400 mb-4" />
                <h3 className="text-lg font-medium text-white">Upload EPUB</h3>
                <p className="text-gray-400 mt-2 mb-6">Drag and drop or click to browse</p>
                <input
                    type="file"
                    accept=".epub"
                    onChange={handleUpload}
                    className="hidden"
                    id="epub-upload"
                />
                <label htmlFor="epub-upload" className="btn-primary">
                    Select File
                </label>
                {uploading && <div className="mt-4 text-violet-400">Uploading and Parsing...</div>}
            </div>

            {chapters.length > 0 && (
                <div className="glass-panel p-6">
                    <div className="flex items-center justify-between mb-4">
                        <h3 className="text-xl font-bold flex items-center gap-2">
                            <Sparkles /> Batch Analysis
                        </h3>
                    </div>
                    <p className="text-gray-400 text-sm mb-4">
                        Analyze all chapters sequentially using LLM to detect characters and emotions.
                    </p>
                    <button
                        onClick={handleAnalyzeAll}
                        disabled={batchProgress.analyzing}
                        className="btn-primary w-full disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {batchProgress.analyzing ? 'Analyzing...' : 'Analyze All Chapters'}
                    </button>
                    {batchProgress.analyzing && (
                        <div className="mt-4">
                            <div className="flex justify-between text-sm text-gray-400 mb-2">
                                <span>{batchProgress.message}</span>
                                <span>{batchProgress.percent}%</span>
                            </div>
                            <div className="w-full bg-gray-700 rounded-full h-2">
                                <div
                                    className="bg-violet-500 h-2 rounded-full transition-all duration-300"
                                    style={{ width: `${batchProgress.percent}%` }}
                                />
                            </div>
                        </div>
                    )}
                </div>
            )}

            {chapters.length > 0 && (
                <div className="glass-panel p-6">
                    <h3 className="text-xl font-bold mb-4 flex items-center gap-2">
                        <FileText /> Chapters
                    </h3>
                    <div className="space-y-2 max-h-[600px] overflow-y-auto pr-2 custom-scrollbar">
                        {chapters.map((ch) => (
                            <button
                                key={ch.id}
                                onClick={() => onSelectChapter(ch)}
                                className="w-full flex justify-between items-center p-4 bg-gray-800/50 rounded-lg hover:bg-gray-800 border border-transparent hover:border-violet-500/50 transition-all group text-left"
                            >
                                <span className="truncate flex-1 font-medium text-lg">{ch.title || `Chapter ${ch.id}`}</span>
                                <ChevronRight className="text-gray-500 group-hover:text-violet-400" />
                            </button>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
