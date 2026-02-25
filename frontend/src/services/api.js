import axios from 'axios';

const API_BASE = 'http://localhost:8080/api';

export const api = {
    getConfig: () => axios.get(`${API_BASE}/config`),
    updateConfig: (config) => axios.post(`${API_BASE}/config`, config),

    uploadEpub: (file) => {
        const formData = new FormData();
        formData.append('file', file);
        return axios.post(`${API_BASE}/upload`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
    },

    analyzeChapter: (chapterId, force = false) => axios.post(`${API_BASE}/analyze/${chapterId}?force=${force}`),
    analyzeAllChapters: (force = false) => axios.post(`${API_BASE}/analyze-all?force=${force}`),

    getCharacters: () => axios.get(`${API_BASE}/characters`),
    confirmMapping: (mapping) => axios.post(`${API_BASE}/confirm-mapping`, mapping),

    // Start generation
    generateAudio: (chapterId) => axios.post(`${API_BASE}/generate/${chapterId}`),
    generateAllAudio: () => axios.post(`${API_BASE}/generate-all`),
    checkAudioStatus: (chapterId) => axios.get(`${API_BASE}/audio-status/${chapterId}`),

    getVoiceList: () => axios.get(`${API_BASE}/voices/list`),
    getVoicePreviewUrl: (path) => `${API_BASE}/voices/preview?path=${encodeURIComponent(path)}`,
    getLLMModels: () => axios.get(`${API_BASE}/llm/models`),
};
