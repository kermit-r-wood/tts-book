import React, { useState } from 'react';
import Settings from './components/Settings';
import ChapterList from './components/ChapterList';
import ChapterDetail from './components/ChapterDetail';
import { BookOpen } from 'lucide-react';

function App() {
    const [activeChapter, setActiveChapter] = useState(null);
    const [chapters, setChapters] = useState([]);
    const [analysisData, setAnalysisData] = useState({}); // { chapterId: [segments] }

    return (
        <div className="layout">
            <header>
                <div className="flex items-center gap-3 text-2xl font-bold text-violet-500">
                    <BookOpen /> TTS Book Creator
                </div>
                {/* No top-level nav in this mode */}
            </header>

            <main>
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    <div className="lg:col-span-2">
                        {!activeChapter ? (
                            <ChapterList
                                chapters={chapters}
                                setChapters={setChapters}
                                onSelectChapter={setActiveChapter}
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
            </main>
        </div>
    );
}

export default App;
