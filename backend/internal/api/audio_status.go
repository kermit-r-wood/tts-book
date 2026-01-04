package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GetAudioStatus checks if audio has been generated for a chapter
func GetAudioStatus(c *gin.Context) {
	chapterID := c.Param("chapterID")

	// Check if audio file exists
	audioPath := fmt.Sprintf("data/out/%s.wav", chapterID)
	_, err := os.Stat(audioPath)

	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"exists":    false,
			"chapterId": chapterID,
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to check audio status: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":    true,
		"chapterId": chapterID,
		"url":       fmt.Sprintf("/output/%s.wav", chapterID),
	})
}
