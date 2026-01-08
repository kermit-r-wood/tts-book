package api

import (
	"net/http"
	"tts-book/backend/internal/config"
	"tts-book/backend/internal/llm"

	"github.com/gin-gonic/gin"
)

func GetConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, cfg)
	}
}

func UpdateConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newCfg config.Config
		if err := c.ShouldBindJSON(&newCfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update values
		cfg.LLMAPIKey = newCfg.LLMAPIKey
		cfg.LLMBaseURL = newCfg.LLMBaseURL
		cfg.LLMModel = newCfg.LLMModel
		cfg.IndexTTSUrl = newCfg.IndexTTSUrl
		cfg.VoiceDir = newCfg.VoiceDir
		cfg.LLMChunkSize = newCfg.LLMChunkSize
		cfg.LLMMinInterval = newCfg.LLMMinInterval
		cfg.MockLLM = newCfg.MockLLM
		cfg.LLMProvider = newCfg.LLMProvider
		cfg.MergeSilence = newCfg.MergeSilence

		// Save
		if err := cfg.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config"})
			return
		}

		c.JSON(http.StatusOK, cfg)
	}
}

func ListLLMModels(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a temporary client with current config to fetch models
		// For now, use stored config.
		client := llm.NewClient(cfg)
		models, err := client.ListModels()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}
