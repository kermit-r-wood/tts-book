package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetCharacters(c *gin.Context) {
	Store.Mu.RLock()
	defer Store.Mu.RUnlock()

	chars := make([]string, 0, len(Store.DetectedCharacters))
	for name := range Store.DetectedCharacters {
		chars = append(chars, name)
	}

	c.JSON(http.StatusOK, gin.H{
		"characters": chars,
		"mapping":    Store.VoiceMapping,
	})
}

func ConfirmMapping(c *gin.Context) {
	var mapping map[string]VoiceConfig
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	Store.Mu.Lock()
	defer Store.Mu.Unlock()

	for name, config := range mapping {
		Store.VoiceMapping[name] = config
	}

	// Persist
	if err := Store.Save(); err != nil {
		log.Printf("Failed to save store: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mapping saved", "count": len(Store.VoiceMapping)})
}
