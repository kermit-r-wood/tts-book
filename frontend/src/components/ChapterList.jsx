import React, { useState } from 'react';
import { api } from '../services/api';
import { Upload, FileText, ChevronRight } from 'lucide-react';

export default function ChapterList({ chapters, setChapters, onSelectChapter }) {
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
