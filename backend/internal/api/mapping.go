package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

type CharacterDetail struct {
	Name     string   `json:"name"`
	Chapters []string `json:"chapters"`
}

func GetCharacters(c *gin.Context) {
	Store.Mu.RLock()
	defer Store.Mu.RUnlock()

	// Helper to track sorting info
	type charInfo struct {
		firstIdx int
		titles   []string
	}
	infoMap := make(map[string]*charInfo)

	// Initialize for all detected characters
	for name := range Store.DetectedCharacters {
		infoMap[name] = &charInfo{firstIdx: -1, titles: []string{}}
	}

	// Iterate chapters to find appearances (preserves order)
	for i, ch := range Store.Chapters {
		segments, ok := Store.Analysis[ch.ID]
		if !ok {
			continue
		}

		// Find unique speakers in this chapter
		speakers := make(map[string]bool)
		for _, seg := range segments {
			if seg.Speaker != "" {
				speakers[seg.Speaker] = true
			}
		}

		// Update info
		for spk := range speakers {
			if info, exists := infoMap[spk]; exists {
				if info.firstIdx == -1 {
					info.firstIdx = i
				}
				info.titles = append(info.titles, ch.Title)
			}
		}
	}

	// Build result slice
	details := make([]CharacterDetail, 0, len(infoMap))
	for name, info := range infoMap {
		details = append(details, CharacterDetail{
			Name:     name,
			Chapters: info.titles,
		})
	}

	// Sort
	sort.Slice(details, func(i, j int) bool {
		idxI := infoMap[details[i].Name].firstIdx
		idxJ := infoMap[details[j].Name].firstIdx

		// Put never-appearing characters at the end
		if idxI == -1 && idxJ == -1 {
			return details[i].Name < details[j].Name
		}
		if idxI == -1 {
			return false
		}
		if idxJ == -1 {
			return true
		}

		if idxI != idxJ {
			return idxI < idxJ
		}
		return details[i].Name < details[j].Name
	})

	c.JSON(http.StatusOK, gin.H{
		"characters": details,
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
