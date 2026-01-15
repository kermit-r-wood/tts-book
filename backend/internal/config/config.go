package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	LLMAPIKey      string `json:"llm_api_key"`
	LLMBaseURL     string `json:"llm_base_url"`
	LLMModel       string `json:"llm_model"`        // Default "ZhipuAI/GLM-4.7"
	LLMProvider    string `json:"llm_provider"`     // "openai" or "gemini"
	LLMChunkSize   int    `json:"llm_chunk_size"`   // Default 1000
	LLMMinInterval int    `json:"llm_min_interval"` // Default 3000 ms
	MockLLM        bool   `json:"mock_llm"`         // Mock LLM responses
	MergeSilence   int    `json:"merge_silence"`    // Silence between audio segments in ms
	NormalizeAudio bool   `json:"normalize_audio"`  // Whether to normalize audio volume
	IndexTTSUrl    string `json:"index_tts_url"`
	VoiceDir       string `json:"voice_dir"`
	Port           string `json:"port"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.Mutex
)

const ConfigFile = "config.json"

func Load() *Config {
	once.Do(func() {
		instance = &Config{
			LLMBaseURL:     "https://api-inference.modelscope.cn/v1", // DeepSeek on ModelScope
			LLMModel:       "ZhipuAI/GLM-4.7",
			LLMProvider:    "openai",
			LLMChunkSize:   800,
			LLMMinInterval: 3000,
			MergeSilence:   400,                     // Default 400ms silence between segments
			NormalizeAudio: true,                    // Default true
			IndexTTSUrl:    "http://127.0.0.1:7860", // Default
			VoiceDir:       "voices",                // Default local voice directory
			Port:           "8080",
		}

		instance.LLMBaseURL = "https://api-inference.modelscope.cn/v1"
		// Try to load from file
		file, err := os.ReadFile(ConfigFile)
		if err == nil {
			_ = json.Unmarshal(file, instance)
		}
	})
	return instance
}

func (c *Config) Save() error {
	mu.Lock()
	defer mu.Unlock()

	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, bytes, 0644)
}

func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}
