package api

import (
	"log"
	"net/http"
	"tts-book/backend/internal/llm"

	"github.com/gin-gonic/gin"
)

type MergeRequest struct {
	Target  string   `json:"target"`
	Sources []string `json:"sources"`
}

type UpdateCharacterRequest struct {
	Name        string      `json:"name"`
	VoiceConfig VoiceConfig `json:"voiceConfig"`
}

// MergeCharacters merges multiple source characters into a single target character
func MergeCharacters(c *gin.Context) {
	var req MergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Target == "" || len(req.Sources) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target and sources are required"})
		return
	}

	Store.Mu.Lock()
	defer Store.Mu.Unlock()

	// 1. Update Analysis Results
	count := 0
	for chapterID, segments := range Store.Analysis {
		updatedSegments := make([]llm.AnalysisResult, len(segments)) // Create new slice to avoid mutating whilst iterating if we were doing that, but here we replace
		copy(updatedSegments, segments)

		changed := false
		for i, seg := range updatedSegments {
			for _, src := range req.Sources {
				if seg.Speaker == src {
					updatedSegments[i].Speaker = req.Target
					changed = true
					count++
					break
				}
			}
		}
		if changed {
			Store.Analysis[chapterID] = updatedSegments
		}
	}

	// 2. Update DetectedCharacters & VoiceMapping
	// Ensure target exists
	Store.DetectedCharacters[req.Target] = true

	// Preserve target voice config if it exists, otherwise try to take from one of the sources?
	// For now, we assume user keeps target's config or sets it later.
	// We just delete sources.

	for _, src := range req.Sources {
		delete(Store.DetectedCharacters, src)
		// We could delete voice mapping, but maybe keep it for reference?
		// No, clean it up.
		delete(Store.VoiceMapping, src)
	}

	// Persist
	if err := Store.Save(); err != nil {
		log.Printf("Failed to save store after merge: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "Merged in memory, but save failed", "count": count})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Characters merged successfully", "count": count})
}

// UpdateCharacter updates voice configuration for a specific character
func UpdateCharacter(c *gin.Context) {
	var req UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Character Name is required"})
		return
	}

	Store.Mu.Lock()
	defer Store.Mu.Unlock()

	Store.VoiceMapping[req.Name] = req.VoiceConfig

	// Also ensure character is in Detected list just in case
	Store.DetectedCharacters[req.Name] = true

	// Persist
	if err := Store.Save(); err != nil {
		log.Printf("Failed to save store after update: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Character updated"})
}
