import React, { useState } from 'react';
import { api } from '../services/api';
import { ArrowLeft, Activity, User, Play, FileText, CheckCircle } from 'lucide-react';
import CharacterStudio from './CharacterStudio';
import Player from './Player';

import { WebSocketClient } from '../services/ws';

export default function ChapterDetail({ chapter, onBack, analysisData, setAnalysisData }) {
    const [activeTab, setActiveTab] = useState('analyze'); // analyze, voices, audio
    const [analyzing, setAnalyzing] = useState(false);
    const [streamOutput, setStreamOutput] = useState('');
    const [generationStarted, setGenerationStarted] = useState(false);

    const hasAnalysis = analysisData[chapter.id];

    // WS Connection for Streaming
    React.useEffect(() => {
        const ws = new WebSocketClient('ws://localhost:8080/api/ws', (msg) => {
            if (msg.type === 'llm_output' && msg.chapterId === chapter.id) {
                setStreamOutput(prev => prev + msg.message);
            }
        });
        ws.connect();
        return () => {
            ws.close();
        };
    }, [chapter.id]);

    const runAnalysis = async (force = false) => {
        setAnalyzing(true);
        setStreamOutput(''); // Reset on new run
        try {
            // Pass force=true if requested
            const res = await api.analyzeChapter(chapter.id, force);
            setAnalysisData(prev => ({ ...prev, [chapter.id]: res.data.results }));
        } catch (e) {
            console.error(e);
            alert('分析失败');
        } finally {
            setAnalyzing(false);
        }
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <button
                    onClick={onBack}
                    className="p-2 hover:bg-white/10 rounded-full transition-colors"
                >
                    <ArrowLeft />
                </button>
                <h2 className="text-2xl font-bold truncate flex-1">{chapter.title}</h2>
            </div>

            {/* Tabs */}
            <div className="flex border-b border-gray-700">
                <button
                    className={`px-6 py-3 font-medium flex items-center gap-2 border-b-2 transition-colors ${activeTab === 'analyze' ? 'border-violet-500 text-violet-400' : 'border-transparent text-gray-400 hover:text-white'}`}
                    onClick={() => setActiveTab('analyze')}
                >
                    <FileText size={18} /> 文本分析
                </button>
                <button
                    className={`px-6 py-3 font-medium flex items-center gap-2 border-b-2 transition-colors ${activeTab === 'voices' ? 'border-violet-500 text-violet-400' : 'border-transparent text-gray-400 hover:text-white'}`}
                    onClick={() => setActiveTab('voices')}
                    disabled={!hasAnalysis}
                >
                    <User size={18} /> 角色配音
                </button>
                <button
                    className={`px-6 py-3 font-medium flex items-center gap-2 border-b-2 transition-colors ${activeTab === 'audio' ? 'border-violet-500 text-violet-400' : 'border-transparent text-gray-400 hover:text-white'}`}
                    onClick={() => setActiveTab('audio')}
                    disabled={!hasAnalysis} // Technically need analysis first? Yes usually.
                >
                    <Play size={18} /> 合成音频
                </button>
            </div>

            {/* Content */}
            <div className="min-h-[500px]">
                {activeTab === 'analyze' && (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        {/* Left: Actions & Status */}
                        <div className="space-y-6">
                            <div className="glass-panel p-6">
                                <h3 className="text-lg font-bold mb-4">分析状态</h3>
                                {hasAnalysis ? (
                                    <div className="flex items-center gap-2 text-green-400 mb-4">
                                        <CheckCircle /> 分析完成
                                    </div>
                                ) : (
                                    <p className="text-gray-400 mb-4">运行分析以识别角色并拆分文本</p>
                                )}

                                <div className="space-y-3">
                                    <button
                                        onClick={() => runAnalysis(false)} // Default: use cache if available
                                        disabled={analyzing}
                                        className={`btn-primary w-full flex justify-center items-center gap-2 ${analyzing ? 'opacity-50' : ''}`}
                                    >
                                        <Activity className={analyzing ? 'animate-spin' : ''} />
                                        {analyzing ? '正在分析...' : (hasAnalysis ? '重新分析 (缓存)' : '开始分析')}
                                    </button>

                                    {hasAnalysis && (
                                        <button
                                            onClick={() => runAnalysis(true)} // Force re-analysis
                                            disabled={analyzing}
                                            className="w-full py-2 text-sm text-red-400 hover:text-red-300 hover:bg-white/5 rounded transition-colors flex justify-center items-center gap-2"
                                        >
                                            强制重新分析 (覆盖)
                                        </button>
                                    )}
                                </div>
                            </div>

                            {/* Text Preview or Stream */}
                            <div className="glass-panel p-6 flex flex-col h-[500px]">
                                <h3 className="text-lg font-bold mb-2 flex justify-between">
                                    <span>{analyzing ? '思考中...' : '原文预览'}</span>
                                    {analyzing && <Activity className="animate-spin text-violet-400" size={18} />}
                                </h3>

                                {analyzing || streamOutput ? (
                                    <div className="flex-1 bg-black/50 rounded-lg p-4 overflow-y-auto custom-scrollbar whitespace-pre-wrap flex flex-col gap-2">
                                        {(() => {
                                            const output = streamOutput || '';
                                            const thinkStart = output.indexOf('<think>');
                                            const thinkEnd = output.indexOf('</think>');

                                            let thinkContent = '';
                                            let mainContent = output;

                                            if (thinkStart !== -1) {
                                                if (thinkEnd !== -1) {
                                                    // Complete think block
                                                    thinkContent = output.substring(thinkStart + 7, thinkEnd);
                                                    mainContent = output.substring(0, thinkStart) + output.substring(thinkEnd + 8);
                                                } else {
                                                    // Open think block (streaming)
                                                    thinkContent = output.substring(thinkStart + 7);
                                                    mainContent = output.substring(0, thinkStart);
                                                }
                                            }

                                            return (
                                                <>
                                                    {thinkContent && (
                                                        <div className="bg-gray-800/50 border-l-2 border-violet-500/50 p-3 rounded text-xs text-gray-400 font-mono italic mb-2">
                                                            <div className="font-bold text-violet-400 mb-1 not-italic">Thinking Process:</div>
                                                            {thinkContent}
                                                        </div>
                                                    )}
                                                    <div className="font-mono text-xs text-green-400">
                                                        {mainContent || (analyzing && !thinkContent && <span className="text-gray-500 animate-pulse">等待大模型响应...</span>)}
                                                    </div>
                                                </>
                                            );
                                        })()}
                                    </div>
                                ) : (
                                    <div className="flex-1 overflow-y-auto text-sm text-gray-400 font-serif leading-relaxed pr-2 custom-scrollbar">
                                        {chapter.content}
                                    </div>
                                )}
                            </div>
                        </div>

                        {/* Right: Results */}
                        <div className="glass-panel p-6 flex flex-col h-full max-h-[700px]">
                            <h3 className="text-lg font-bold mb-4 flex items-center gap-2">
                                <Activity size={18} /> 分析结果
                            </h3>
                            {hasAnalysis ? (
                                <div className="space-y-3 overflow-y-auto flex-1 pr-2 custom-scrollbar">
                                    {analysisData[chapter.id].map((seg, idx) => (
                                        <div key={idx} className="p-3 bg-gray-800/40 rounded border border-gray-700/50 text-sm">
                                            <div className="flex justify-between items-center mb-1">
                                                <span className={`text-xs uppercase font-bold px-1.5 py-0.5 rounded ${seg.speaker === 'Narrator' ? 'bg-gray-700 text-gray-300' : 'bg-violet-900/50 text-violet-300'}`}>
                                                    {seg.speaker}
                                                </span>
                                                <span className="text-xs text-gray-500">{seg.emotion}</span>
                                            </div>
                                            <p className="text-gray-300 mb-1">{seg.text}</p>
                                            {// Display typesetting if it exists and is different from text (or always, depending on user need. 
                                                // User asked to 'show typesetting', likely to check pinyin.)
                                                seg.typesetting && (
                                                    <div className="mt-1 text-xs text-gray-500 font-mono border-t border-gray-700/50 pt-1">
                                                        <span className="select-none text-gray-600 mr-2">Typesetting:</span>
                                                        <span className="text-violet-300">{seg.typesetting}</span>
                                                    </div>
                                                )
                                            }
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className="flex-1 flex items-center justify-center text-gray-500 italic">
                                    暂无分析数据
                                </div>
                            )}
                        </div>
                    </div>
                )}

                <div className={activeTab === 'voices' ? 'block' : 'hidden'}>
                    <CharacterStudio
                        chapterId={chapter.id}
                        analysisData={analysisData}
                        onGenerate={() => {
                            setGenerationStarted(true);
                            setActiveTab('audio');
                        }}
                        embedded={true}
                    />
                </div>

                <div className={activeTab === 'audio' ? 'block' : 'hidden'}>
                    {(activeTab === 'audio' || generationStarted) && (
                        <Player
                            chapterId={chapter.id}
                        />
                    )}
                </div>
            </div>
        </div>
    );
}
