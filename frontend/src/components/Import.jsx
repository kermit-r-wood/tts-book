import React, { useState } from 'react';
import { api } from '../services/api';
import { Upload, FileText, Play, Activity } from 'lucide-react';

export default function Import({ chapters, setChapters, analysisData, setAnalysisData, onChapterSelect }) {
    // const [chapters, setChapters] = useState([]); // Lifted to App.js
    const [uploading, setUploading] = useState(false);

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

    const [analyzingId, setAnalyzingId] = useState(null);

    const analyze = async (chapterId) => {
        if (analyzingId) return; // Prevent multiple
        setAnalyzingId(chapterId);
        try {
            const res = await api.analyzeChapter(chapterId);
            setAnalysisData(prev => ({ ...prev, [chapterId]: res.data.results }));
            onChapterSelect(chapterId, 'analyze');
        } catch (e) {
            console.error(e);
            alert('Analysis failed: ' + (e.response?.data?.error || e.message));
        } finally {
            setAnalyzingId(null);
        }
    };

    const [viewingChapter, setViewingChapter] = useState(null);

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
                    <h3 className="text-xl font-bold mb-4 flex items-center gap-2">
                        <FileText /> Chapters
                    </h3>
                    <div className="space-y-2 max-h-[500px] overflow-y-auto pr-2 custom-scrollbar">
                        {chapters.map((ch) => (
                            <div key={ch.id} className="flex justify-between items-center p-3 bg-gray-800/50 rounded-lg hover:bg-gray-800 transition-colors">
                                <span className="truncate flex-1 font-medium">{ch.title || `Chapter ${ch.id}`}</span>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setViewingChapter(ch)}
                                        className="p-2 text-gray-400 hover:text-white bg-gray-700/50 rounded-lg hover:bg-gray-700"
                                        title="View Text"
                                    >
                                        <FileText size={18} />
                                    </button>
                                    <button
                                        onClick={() => analyze(ch.id)}
                                        disabled={analyzingId !== null}
                                        className={`p-2 rounded-lg flex items-center gap-2 transition-all ${analyzingId === ch.id
                                            ? 'bg-violet-500 text-white animate-pulse cursor-wait'
                                            : 'text-violet-400 hover:text-violet-300 bg-violet-500/10 hover:bg-violet-500/20'
                                            } ${analyzingId && analyzingId !== ch.id ? 'opacity-50 cursor-not-allowed' : ''}`}
                                        title="Analyze & Voice Map"
                                    >
                                        <Activity size={18} className={analyzingId === ch.id ? 'animate-spin' : ''} />
                                        {analyzingId === ch.id ? 'Processing...' : 'Analyze'}
                                    </button>
                                    {/* Generate button visible only if verified? For now simple flow */}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {viewingChapter && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="glass-panel w-full max-w-4xl max-h-[85vh] flex flex-col p-0 overflow-hidden shadow-2xl border-violet-500/30">
                        <div className="p-4 border-b border-gray-700 flex justify-between items-center bg-gray-900/50">
                            <h3 className="text-xl font-bold text-white truncate">{viewingChapter.title}</h3>
                            <button
                                onClick={() => setViewingChapter(null)}
                                className="text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg px-3 py-1 transition-colors"
                            >
                                Close
                            </button>
                        </div>
                        <div className="p-6 overflow-y-auto font-serif text-lg leading-relaxed text-gray-200 bg-gray-900/30 whitespace-pre-wrap">
                            {viewingChapter.content}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
