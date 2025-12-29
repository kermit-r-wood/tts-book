package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"tts-book/backend/internal/api"
	"tts-book/backend/internal/config"
	"tts-book/backend/internal/epub"
	"tts-book/backend/internal/llm"

	"github.com/gin-gonic/gin"
)

func TestAnalyzeChapter_MockMode(t *testing.T) {
	// 1. Setup Config (Directly modify singleton)
	cfg := config.Get()
	cfg.MockLLM = true
	cfg.LLMAPIKey = "dummy_key"
	t.Logf("Test Config: %+v", cfg)
	t.Logf("Global Config via Get(): %+v", config.Get())

	// 2. Setup Gin
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api.SetupRoutes(r, cfg)

	// 3. Preload a dummy chapter into memory store
	chapterID := "ch_test"
	dummyChapters := []epub.Chapter{
		{
			ID:      chapterID,
			Title:   "Test Chapter",
			Content: "This is a mock narration segment. This is a mock dialogue segment. Another mock narration.",
		},
	}
	// Manually inject into internal store var (exported for test or we just use Upload mechanism...
	// but Upload requires file. Let's direct inject to LoadedChapters if accessible.
	// Wait, api.LoadedChapters is exported? Yes from my grep earlier.)
	api.LoadedChapters["current"] = dummyChapters

	// SETUP: Mock Book ID to test per-book persistence
	mockBookID := "mock_book_123"
	api.Store.Mu.Lock()
	api.Store.BookID = mockBookID
	api.Store.Mu.Unlock()
	defer func() {
		// Cleanup specific file
		os.Remove(fmt.Sprintf("data/%s.json", mockBookID))
	}()

	// Perform Request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/analyze/ch_test", nil)
	r.ServeHTTP(w, req)

	// Assertions
	t.Logf("Response Code: %d", w.Code)
	t.Logf("Response Body: %s", w.Body.String())

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		ChapterID string               `json:"chapterId"`
		Results   []llm.AnalysisResult `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ChapterID != chapterID {
		t.Errorf("Expected chapterId %s, got %s", chapterID, response.ChapterID)
	}
	if len(response.Results) == 0 {
		t.Error("Expected results, got empty list")
	}

	t.Logf("Results type: %T", response.Results)

	// Check Backend Store State
	api.Store.Mu.RLock()
	stored, ok := api.Store.Analysis[chapterID]
	api.Store.Mu.RUnlock()

	if !ok {
		t.Error("Store was not updated with analysis results")
	}
	if len(stored) != len(response.Results) {
		t.Errorf("Store has %d items, response has %d", len(stored), len(response.Results))
	}

	// Verify JSON file created for this book
	expectedFile := fmt.Sprintf("data/%s.json", mockBookID)
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected persistence file %s to be created, but it was not found", expectedFile)
	} else {
		t.Logf("Verified persistence file created: %s", expectedFile)
	}
}
