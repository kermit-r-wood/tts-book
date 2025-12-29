package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
)

type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "dir" or "file"
	Path string `json:"path"`
}

// BrowseFiles lists files in a directory
func BrowseFiles(c *gin.Context) {
	dirPath := c.Query("path")
	if dirPath == "" {
		// Default to current directory
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			dirPath = "."
		}
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var files []FileEntry
	var dirs []FileEntry

	// Add ".." entry if not at root
	parent := filepath.Dir(dirPath)
	if parent != dirPath {
		dirs = append(dirs, FileEntry{Name: "..", Type: "dir", Path: parent})
	}

	for _, e := range entries {
		// Skip hidden files
		if e.Name()[0] == '.' {
			continue
		}

		fullPath := filepath.Join(dirPath, e.Name())
		entry := FileEntry{
			Name: e.Name(),
			Path: fullPath,
		}

		if e.IsDir() {
			entry.Type = "dir"
			dirs = append(dirs, entry)
		} else {
			entry.Type = "file"
			// Filter audio extensions?
			ext := filepath.Ext(e.Name())
			if ext == ".wav" || ext == ".mp3" || ext == ".ogg" || ext == ".flac" {
				files = append(files, entry)
			}
		}
	}

	// Sort
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	result := append(dirs, files...)

	c.JSON(http.StatusOK, gin.H{
		"current": dirPath,
		"entries": result,
	})
}
