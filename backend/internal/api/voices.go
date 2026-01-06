package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tts-book/backend/internal/config"

	"github.com/gin-gonic/gin"
)

// UploadVoice handles uploading a reference audio file
func UploadVoice(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Create voices directory if not exists
	uploadDir := "data/voices"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Generate unique filename to avoid collision
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Return absolute path so TTS can use it locally
	absPath, err := filepath.Abs(dst)
	if err != nil {
		// Fallback to relative
		absPath = dst
	}

	c.JSON(http.StatusOK, gin.H{
		"path":     absPath,
		"filename": filename,
	})
}

// GetVoicesFromDir returns a list of audio files in the directory
func GetVoicesFromDir(dirPath string) ([]string, error) {
	var voices []string
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".wav" || ext == ".mp3" || ext == ".ogg" || ext == ".flac" {
			voices = append(voices, filepath.Join(dirPath, e.Name()))
		}
	}
	return voices, nil
}

// ListConfiguredVoices returns the list of voices in the configured directory
func ListConfiguredVoices(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.VoiceDir == "" {
			c.JSON(http.StatusOK, gin.H{"voices": []string{}})
			return
		}

		voices, err := GetVoicesFromDir(cfg.VoiceDir)
		if err != nil {
			// Don't fail hard, just return empty and maybe log
			// For user feedback, maybe we should return error
			c.JSON(http.StatusOK, gin.H{"voices": []string{}, "error": err.Error()})
			return
		}

		type VoiceOption struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		var options []VoiceOption
		for _, v := range voices {
			absPath, err := filepath.Abs(v)
			if err != nil {
				absPath = v // Fallback
			}
			options = append(options, VoiceOption{
				Name: filepath.Base(v),
				Path: absPath,
			})
		}
		c.JSON(http.StatusOK, gin.H{"voices": options})
	}
}

// PreviewVoice serves the audio file for preview
func PreviewVoice(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	// Basic security check: ensure it exists and is a file
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info.IsDir() {
		fmt.Printf("[PreviewVoice] File not found or is dir: %s\n", path)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	fmt.Printf("[PreviewVoice] Serving file: %s\n", path)

	// Set content type
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".wav":
		c.Header("Content-Type", "audio/wav")
	case ".mp3":
		c.Header("Content-Type", "audio/mpeg")
	case ".ogg":
		c.Header("Content-Type", "audio/ogg")
	case ".flac":
		c.Header("Content-Type", "audio/flac")
	}

	// Serve the file
	c.File(path)
}
