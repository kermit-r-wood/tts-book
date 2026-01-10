package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"tts-book/backend/internal/epub"
	"tts-book/backend/internal/llm"
)

// ProjectStore holds the state of the current loaded book and analysis
type ProjectStore struct {
	Mu sync.RWMutex

	// Book ID (MD5 Hash)
	BookID string

	CurrentBookPath string
	Chapters        []epub.Chapter

	// ChapterID -> List of Analysis Results (Text segments with generic Speaker)
	Analysis map[string][]llm.AnalysisResult

	// Set of all unique characters found
	DetectedCharacters map[string]bool

	// Character Name -> Voice Config
	VoiceMapping map[string]VoiceConfig
}

type VoiceConfig struct {
	VoiceID       string  `json:"voiceId"`
	Emotion       string  `json:"emotion"`       // Default emotion
	UseLLMEmotion bool    `json:"useLLMEmotion"` // If true, use emotion from LLM analysis; if false, use default emotion
	Speed         float64 `json:"speed"`
	RefAudio      string  `json:"refAudio"` // Path to reference audio for cloning
}

var Store = &ProjectStore{
	Analysis:           make(map[string][]llm.AnalysisResult),
	DetectedCharacters: make(map[string]bool),
	VoiceMapping:       make(map[string]VoiceConfig),
}

func (s *ProjectStore) getStorePath() string {
	if s.BookID == "" {
		return "data/store.json" // Default/Fallback
	}
	return filepath.Join("data", s.BookID+".json")
}

func (s *ProjectStore) Save() error {
	// NOTE: Caller must hold the lock (either RLock or Lock)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	path := s.getStorePath()

	// Ensure dir exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (s *ProjectStore) Load(bookID string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	s.BookID = bookID
	path := s.getStorePath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Reset state for new book
		s.Analysis = make(map[string][]llm.AnalysisResult)
		s.DetectedCharacters = make(map[string]bool)
		s.VoiceMapping = make(map[string]VoiceConfig)
		s.CurrentBookPath = ""
		// Chapters kept? No, Chapters are loaded from memory in UploadEPUB usually.
		// Actually, Store holds Chapters too? Yes.
		// But UploadEPUB populates them immediately after this Load call maybe?
		// If we load existing, we want 'Chapters' from disk too if we saved them?
		// The current Store has 'Chapters' array.
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, s)
}
