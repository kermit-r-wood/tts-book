package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"tts-book/backend/internal/epub"

	"github.com/gin-gonic/gin"
)

// In-memory store for demo purposes. Ideally use a DB or persistent file store.
var LoadedChapters map[string][]epub.Chapter
var CurrentBookPath string

func init() {
	LoadedChapters = make(map[string][]epub.Chapter)
}

func UploadEPUB(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Save uploaded file
	// Ensure upload dir exists
	uploadDir := "uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	dst := filepath.Join(uploadDir, file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Parse EPUB
	reader, err := epub.NewReader(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse EPUB: %v", err)})
		return
	}

	chapters, err := reader.GetChapters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to extract chapters: %v", err)})
		return
	}

	// Calculate MD5 of the file to use as BookID
	f, err := os.Open(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open saved file for hashing"})
		return
	}
	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		f.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate hash"})
		return
	}
	f.Close()

	bookID := hex.EncodeToString(hasher.Sum(nil))

	// Load existing data for this book if available
	if err := Store.Load(bookID); err != nil {
		fmt.Printf("Warning: Failed to load store for book %s: %v\n", bookID, err)
	}

	// Update Store
	Store.Mu.Lock()
	Store.CurrentBookPath = dst
	Store.Chapters = chapters // Store chapters in struct too
	// Note: Analysis, VoiceMapping are preserved if loaded, or empty if new
	Store.Mu.Unlock()

	// Also keep the global singleton for now if legacy code uses it (CurrentBookPath var in upload.go)
	CurrentBookPath = dst // This local var might be redundant now, but keep for safety
	LoadedChapters["current"] = chapters

	// Persist immediately to ensure BookID is saved/file created
	if err := Store.Save(); err != nil {
		fmt.Printf("Warning: Failed to save initial store: %v\n", err)
	}

	// Return stripped chapters (maybe without full content to save bandwidth?)
	// For now return full.
	c.JSON(http.StatusOK, gin.H{
		"message":  "Upload successful",
		"chapters": chapters,
		"bookPath": dst,
	})
}
