import React, { useState } from 'react';
import Settings from './components/Settings';
import ChapterList from './components/ChapterList';
import ChapterDetail from './components/ChapterDetail';
import CharacterManagement from './components/CharacterManagement';
import { BookOpen, Users } from 'lucide-react';

function App() {
    const [view, setView] = useState('chapters'); // 'chapters' | 'characters'
    const [activeChapter, setActiveChapter] = useState(null);
    const [chapters, setChapters] = useState([]);
    const [analysisData, setAnalysisData] = useState({}); // { chapterId: [segments] }
    const [batchProgress, setBatchProgress] = useState({ percent: 0, message: '', analyzing: false });

    // When switching back to chapters, we might want to refresh data if merges happened.
    // However, existing simple state might be stale. AnalysisData is centralized here!
    // If Backend `characters/merge` updates file on disk, but `analysisData` in memory is old,
    // we need to invalidate `analysisData`.
    // Let's clear `analysisData` when coming back from 'characters' view to force re-fetch if needed (or just let ChapterDetail fetch).
    // Actually ChapterDetail uses `analysisData` prop but also fetches?
    // Let's check ChapterDetail later. For now, let's just allow switching.

    const handleViewChange = (newView) => {
        if (newView === 'chapters' && view === 'characters') {
            // Potentially invalidate/refresh if we did merges. Simple way:
            setAnalysisData({});
        }
        setView(newView);
        setActiveChapter(null);
    };

    return (
        <div className="layout">
            <header className="flex justify-between items-center pr-6">
                <div className="flex items-center gap-3 text-2xl font-bold text-violet-500">
                    <BookOpen /> 有声书制作工具
                </div>
                <nav className="flex gap-2">
                    <button
                        onClick={() => handleViewChange('chapters')}
                        className={`px-4 py-2 rounded-lg font-medium transition-colors ${view === 'chapters' ? 'bg-violet-100 text-violet-700' : 'text-slate-500 hover:bg-slate-50'}`}
                    >
                        Chapters
                    </button>
                    <button
                        onClick={() => handleViewChange('characters')}
                        className={`px-4 py-2 rounded-lg font-medium transition-colors flex items-center gap-2 ${view === 'characters' ? 'bg-violet-100 text-violet-700' : 'text-slate-500 hover:bg-slate-50'}`}
                    >
                        <Users size={18} /> Characters
                    </button>
                </nav>
            </header>

            <main>
                {view === 'characters' ? (
                    <CharacterManagement onBack={() => handleViewChange('chapters')} />
                ) : (
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        <div className="lg:col-span-2">
                            {!activeChapter ? (
                                <ChapterList
                                    chapters={chapters}
                                    setChapters={setChapters}
                                    onSelectChapter={setActiveChapter}
                                    batchProgress={batchProgress}
                                    setBatchProgress={setBatchProgress}
                                />
                            ) : (
                                <ChapterDetail
                                    chapter={activeChapter}
                                    onBack={() => setActiveChapter(null)}
                                    analysisData={analysisData}
                                    setAnalysisData={setAnalysisData}
                                />
                            )}
                        </div>

                        <div>
                            <Settings />
                        </div>
                    </div>
                )}
            </main>
        </div>
    );
}

export default App;
