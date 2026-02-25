package api

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"math/rand"
	"path/filepath"
	"time"

	// Added for time.Sleep
	"tts-book/backend/internal/audio"
	"tts-book/backend/internal/config"
	"tts-book/backend/internal/llm"
	"tts-book/backend/internal/tts"

	"github.com/gin-gonic/gin"
)

// GenerateAudio orchestrates the TTS generation
func GenerateAudio(c *gin.Context) {
	chapterID := c.Param("chapterID")

	Store.Mu.RLock()
	segments, ok := Store.Analysis[chapterID]
	bookID := Store.BookID
	// mapping := Store.VoiceMapping // Copy if needed
	Store.Mu.RUnlock()

	if !ok || len(segments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chapter not analyzed yet"})
		return
	}

	// Start async generation
	go func() {
		BroadcastProgress(chapterID, 0, "Initializing TTS...")

		// Initialize TTS Client
		cfg := config.Get()
		ttsClient := tts.NewClient(cfg)

		// Prepare temp dir
		tempDir := fmt.Sprintf("data/temp/%s", chapterID)
		os.MkdirAll(tempDir, 0755)

		outDir := fmt.Sprintf("data/out/%s", bookID)
		os.MkdirAll(outDir, 0755) // Ensure output dir exists

		total := len(segments)
		var filePaths []string

		for i, seg := range segments {
			// Determine voice/emotion from mapping
			Store.Mu.RLock()
			mapping, hasMapping := Store.VoiceMapping[seg.Speaker]
			Store.Mu.RUnlock()

			voice := ""
			emotion := seg.Emotion

			if hasMapping {
				voice = mapping.VoiceID // Use full path for internal use

				// Determine emotion based on UseLLMEmotion setting

				// Determine emotion based on UseLLMEmotion setting
				// Default to true if nil
				useLLM := true
				if mapping.UseLLMEmotion != nil {
					useLLM = *mapping.UseLLMEmotion
				}

				if useLLM {

					// Use emotion from LLM analysis (seg.Emotion)

					if seg.Emotion != "" {

						emotion = seg.Emotion

					}

				} else {

					// Use default emotion from mapping

					if mapping.Emotion != "" {

						emotion = mapping.Emotion

					}

				}

			}

			if emotion == "" {

				emotion = "calm"

			}

			if emotion == "" {
				emotion = "calm"
			}

			// Generate
			// Generate
			runes := []rune(seg.Text)
			display := string(runes)
			if len(runes) > 10 {
				display = string(runes[:10])
			}
			BroadcastProgress(chapterID, int((float64(i)/float64(total))*100), fmt.Sprintf("Generating (%d/%d): %s...", i+1, total, display))

			textToSpeak := seg.Typesetting
			if textToSpeak == "" {
				textToSpeak = seg.Text
			}

			// Check for length and split if necessary
			maxChars := 20 // Hardcoded limit for now
			chunks := SplitTextForTTS(textToSpeak, maxChars)

			var segmentAudioData []byte

			if len(chunks) == 1 {
				// Original logic for single chunk
				log.Printf("[TTS] Generating segment %d with text: %s", i, textToSpeak)
				audioData, err := ttsClient.Generate(textToSpeak, voice, emotion)
				if err != nil {
					log.Printf("[TTS] Error generating segment %d: %v", i, err)
					BroadcastProgress(chapterID, 0, fmt.Sprintf("Error: %v", err))
					return
				}
				segmentAudioData = audioData
			} else {
				// Split logic
				log.Printf("[TTS] Segment %d is long (%d chars), split into %d chunks", i, len([]rune(textToSpeak)), len(chunks))
				var chunkFiles []string

				for j, chunk := range chunks {
					BroadcastProgress(chapterID, int((float64(i)/float64(total))*100), fmt.Sprintf("Generating (%d/%d): Part %d/%d...", i+1, total, j+1, len(chunks)))
					log.Printf("[TTS] Generating segment %d chunk %d: %s", i, j, chunk)

					chunkAudio, err := ttsClient.Generate(chunk, voice, emotion)
					if err != nil {
						log.Printf("[TTS] Error generating segment %d chunk %d: %v", i, j, err)
						BroadcastProgress(chapterID, 0, fmt.Sprintf("Error in chunk: %v", err))
						return
					}

					// Save chunk to temp
					chunkPath := fmt.Sprintf("%s/%d_part_%d.wav", tempDir, i, j)
					if err := os.WriteFile(chunkPath, chunkAudio, 0644); err != nil {
						log.Printf("[TTS] Failed to write chunk file: %v", err)
						return
					}
					chunkFiles = append(chunkFiles, chunkPath)
				}

				// Merge chunks into one segment
				// Using 0ms silence for intra-segment merge, as splits might be comma-based.
				// If we want sentence pauses, we'd need smarter splitting or rely on TTS natural pause.
				mergedPath := fmt.Sprintf("%s/%d_merged.wav", tempDir, i)
				if err := audio.MergeAndNormalize(chunkFiles, mergedPath, 0, false); err != nil {
					log.Printf("[TTS] Failed to merge chunks for segment %d: %v", i, err)
					return
				}

				// Read back the merged data to be consistent with main flow
				// (Though efficiently we could just rename it to the final destination, but existing logic appends to filePaths)
				data, err := os.ReadFile(mergedPath)
				if err != nil {
					log.Printf("[TTS] Failed to read merged segment %d: %v", i, err)
					return
				}
				segmentAudioData = data

				// Cleanup chunk files
				for _, f := range chunkFiles {
					os.Remove(f)
				}
				os.Remove(mergedPath) // Will be rewritten below as final segment file
			}

			// Save temp file (final segment)
			filePath := fmt.Sprintf("%s/%d.wav", tempDir, i)
			if err := os.WriteFile(filePath, segmentAudioData, 0644); err != nil {
				log.Printf("[TTS] Failed to write temp file: %v", err)
				return
			}
			filePaths = append(filePaths, filePath)
		}

		// Merge
		BroadcastProgress(chapterID, 95, "Merging Audio Files...")
		outPath := fmt.Sprintf("%s/%s.wav", outDir, chapterID)

		if err := audio.MergeAndNormalize(filePaths, outPath, cfg.MergeSilence, cfg.NormalizeAudio); err != nil {
			log.Printf("[TTS] Merge failed: %v", err)
			BroadcastProgress(chapterID, 0, fmt.Sprintf("Merge Error: %v", err))
			return
		}

		// Cleanup temp
		os.RemoveAll(tempDir)

		BroadcastProgress(chapterID, 100, "Generation Complete!")
	}()

	c.JSON(http.StatusOK, gin.H{"status": "started", "chapterId": chapterID})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func AnalyzeChapter(c *gin.Context) {
	chapterID := c.Param("chapterID")
	force := c.Query("force") == "true"

	// Check if already analyzed (and not forcing re-analysis)
	Store.Mu.RLock()
	existing, exists := Store.Analysis[chapterID]
	Store.Mu.RUnlock()

	if exists && len(existing) > 0 && !force {
		log.Printf("[Analyze] Returning cached analysis for chapter %s", chapterID)
		c.JSON(http.StatusOK, gin.H{
			"chapterId": chapterID,
			"results":   existing,
			"cached":    true, // Optional: let frontend know it was cached
		})
		return
	}

	// Retrieve chapter content from memory store
	// For MVP, we iterate LoadedChapters["current"] to find ID
	chapters, ok := LoadedChapters["current"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No book loaded"})
		return
	}

	var textToAnalyze string
	for _, ch := range chapters {
		if ch.ID == chapterID {
			textToAnalyze = ch.Content
			break
		}
	}

	if textToAnalyze == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chapter not found or empty"})
		return
	}

	// Initialize LLM Client
	cfg := config.Get()
	if cfg.LLMAPIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "LLM API Key is missing in config"})
		return
	}

	client := llm.NewClient(cfg)

	// Chunking text to prevent LLM context issues (losing narrators)
	// Limit based on config
	limit := cfg.LLMChunkSize
	if limit <= 0 {
		limit = 1000
	}
	chunks := SplitText(textToAnalyze, limit)
	log.Printf("[Analyze] Split Chapter %s into %d chunks (limit: %d)\n", chapterID, len(chunks), limit)

	var allResults []llm.AnalysisResult

	for i, chunk := range chunks {
		log.Printf("[Analyze] Processing chunk %d/%d (len: %d)\n", i+1, len(chunks), len(chunk))

		results, err := client.AnalyzeTextStream(chunk, func(token string) {
			BroadcastLLMOutput(chapterID, token)
		})
		if err != nil {
			log.Printf("[Analyze] Chunk %d failed: %v\n", i+1, err)
			// Continue with best effort? Or fail?
			// Let's log and likely continue to get partial results, but appending nothing for this chunk.
			// Or better, error out to let user know something went wrong.
			// Given "prevent this behavior", reliability is key. But one chunk fail shouldn't necessarily kill everything if others worked.
			// However, to be safe, let's treat it as a task failure for now so user can retry.
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Analysis failed at chunk %d: %v", i+1, err)})
			return
		}

		allResults = append(allResults, results...)
	}

	log.Printf("[Analyze] Success. Found total %d segments.\n", len(allResults))

	// Save results to Store
	// Auto-assign voices to new characters
	voiceDir := cfg.VoiceDir
	if voiceDir == "" {
		voiceDir = "voices" // Fallback if config is empty
	}
	availableVoices, err := GetVoicesFromDir(voiceDir)
	if err != nil {
		log.Printf("[Analyze] Warning: Could not list voices from %s: %v", voiceDir, err)
	}

	// Shuffle voices for random assignment
	if len(availableVoices) > 0 {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rnd.Shuffle(len(availableVoices), func(i, j int) {
			availableVoices[i], availableVoices[j] = availableVoices[j], availableVoices[i]
		})
	}

	Store.Mu.Lock()
	Store.Analysis[chapterID] = allResults

	nextVoiceIdx := 0
	for _, r := range allResults {
		if r.Speaker != "" {
			Store.DetectedCharacters[r.Speaker] = true

			// Assign voice if character doesn't have one OR has one with empty VoiceID
			config, exists := Store.VoiceMapping[r.Speaker]
			if (!exists || config.VoiceID == "") && len(availableVoices) > 0 {
				// Pick next voice
				voicePath := availableVoices[nextVoiceIdx%len(availableVoices)]
				nextVoiceIdx++

				Store.VoiceMapping[r.Speaker] = VoiceConfig{
					VoiceID:       voicePath,
					Emotion:       "calm",
					UseLLMEmotion: boolPtr(true), // Default to using LLM emotions
				}
				log.Printf("[Analyze] Auto-assigned voice %s to character %s", filepath.Base(voicePath), r.Speaker)
			}
		}
	}
	Store.Mu.Unlock()

	// Persist
	if err := Store.Save(); err != nil {
		log.Printf("[Analyze] Warning: Failed to save store: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"chapterId": chapterID,
		"results":   allResults,
	})
}

// AnalyzeAllChapters analyzes all chapters sequentially
func AnalyzeAllChapters(c *gin.Context) {
	force := c.Query("force") == "true"

	// Get all chapters
	chapters, ok := LoadedChapters["current"]
	if !ok || len(chapters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No book loaded"})
		return
	}

	// Start async batch analysis
	go func() {
		total := len(chapters)
		successCount := 0
		failedChapters := []string{}

		for i, chapter := range chapters {
			chapterID := chapter.ID
			chapterTitle := chapter.Title
			if chapterTitle == "" {
				chapterTitle = fmt.Sprintf("Chapter %s", chapterID)
			}

			// Broadcast overall progress
			overallPercent := int((float64(i) / float64(total)) * 100)
			BroadcastProgress("batch", overallPercent, fmt.Sprintf("Analyzing %s (%d/%d)...", chapterTitle, i+1, total))

			// Check if already analyzed (and not forcing re-analysis)
			Store.Mu.RLock()
			existing, exists := Store.Analysis[chapterID]
			Store.Mu.RUnlock()

			if exists && len(existing) > 0 && !force {
				log.Printf("[AnalyzeAll] Skipping already analyzed chapter %s", chapterID)
				successCount++
				continue
			}

			// Analyze this chapter
			textToAnalyze := chapter.Content
			if textToAnalyze == "" {
				log.Printf("[AnalyzeAll] Chapter %s is empty, skipping", chapterID)
				failedChapters = append(failedChapters, chapterTitle)
				continue
			}

			// Initialize LLM Client
			cfg := config.Get()
			if cfg.LLMAPIKey == "" {
				log.Printf("[AnalyzeAll] LLM API Key is missing")
				BroadcastProgress("batch", 0, "Error: LLM API Key is missing")
				return
			}

			client := llm.NewClient(cfg)

			// Chunking text
			limit := cfg.LLMChunkSize
			if limit <= 0 {
				limit = 1000
			}
			chunks := SplitText(textToAnalyze, limit)
			log.Printf("[AnalyzeAll] Chapter %s split into %d chunks\n", chapterID, len(chunks))

			var allResults []llm.AnalysisResult
			chunkFailed := false

			for j, chunk := range chunks {
				log.Printf("[AnalyzeAll] Processing chapter %s chunk %d/%d\n", chapterID, j+1, len(chunks))

				results, err := client.AnalyzeTextStream(chunk, func(token string) {
					BroadcastLLMOutput(chapterID, token)
				})
				if err != nil {
					log.Printf("[AnalyzeAll] Chapter %s chunk %d failed: %v\n", chapterID, j+1, err)
					failedChapters = append(failedChapters, chapterTitle)
					chunkFailed = true
					break
				}

				allResults = append(allResults, results...)
			}

			if chunkFailed {
				continue
			}

			log.Printf("[AnalyzeAll] Chapter %s analyzed successfully. Found %d segments.\n", chapterID, len(allResults))

			// Auto-assign voices to new characters
			voiceDir := cfg.VoiceDir
			if voiceDir == "" {
				voiceDir = "voices"
			}
			availableVoices, err := GetVoicesFromDir(voiceDir)
			if err != nil {
				log.Printf("[AnalyzeAll] Warning: Could not list voices from %s: %v", voiceDir, err)
			}

			// Shuffle voices for random assignment
			if len(availableVoices) > 0 {
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				rnd.Shuffle(len(availableVoices), func(i, j int) {
					availableVoices[i], availableVoices[j] = availableVoices[j], availableVoices[i]
				})
			}

			Store.Mu.Lock()
			Store.Analysis[chapterID] = allResults

			nextVoiceIdx := 0
			for _, r := range allResults {
				if r.Speaker != "" {
					Store.DetectedCharacters[r.Speaker] = true

					// Assign voice if character doesn't have one OR has one with empty VoiceID
					config, exists := Store.VoiceMapping[r.Speaker]
					if (!exists || config.VoiceID == "") && len(availableVoices) > 0 {
						// Pick next voice
						voicePath := availableVoices[nextVoiceIdx%len(availableVoices)]
						nextVoiceIdx++

						Store.VoiceMapping[r.Speaker] = VoiceConfig{
							VoiceID: voicePath,

							Emotion: "calm",

							UseLLMEmotion: boolPtr(true), // Default to using LLM emotions
						}

					}
				}
			}
			Store.Mu.Unlock()

			// Persist after each chapter
			if err := Store.Save(); err != nil {
				log.Printf("[AnalyzeAll] Warning: Failed to save store: %v", err)
			}

			successCount++
		}

		// Broadcast completion
		if len(failedChapters) > 0 {
			BroadcastProgress("batch", 100, fmt.Sprintf("Completed with errors. %d/%d chapters analyzed. Failed: %s", successCount, total, failedChapters))
		} else {
			BroadcastProgress("batch", 100, fmt.Sprintf("All chapters analyzed successfully! (%d/%d)", successCount, total))
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "started", "totalChapters": len(chapters)})
}

// SplitText splits a large string into chunks strictly less than limit.
// It tries to split at paragraph boundaries (\n\n), then sentences (.), then arbitrary.
func SplitText(text string, limit int) []string {
	if len(text) <= limit {
		return []string{text}
	}

	var chunks []string
	runes := []rune(text)
	length := len(runes)

	start := 0
	for start < length {
		end := start + limit
		if end >= length {
			chunks = append(chunks, string(runes[start:]))
			break
		}

		// Look for best split point
		// 1. Try \n\n (Paragraphs) within the last 20% of the chunk window to ensure we don't stick to the very end
		// Actually, just look backwards from 'end'

		foundSplit := false
		splitIdx := -1

		// Look backwards for \n\n
		searchLimit := 1000 // How far back to search for ideal break
		if searchLimit > limit {
			searchLimit = limit / 2
		}

		// Priority 1: Double Newline
		for i := end; i > start+limit-searchLimit && i > start; i-- {
			if i+1 < length && runes[i] == '\n' && runes[i-1] == '\n' {
				splitIdx = i + 1 // Include the newlines in the previous chunk or next?
				// Usually split after newlines.
				splitIdx = i + 1
				foundSplit = true
				break
			}
		}

		// Priority 2: Single Newline
		if !foundSplit {
			for i := end; i > start+limit-searchLimit && i > start; i-- {
				if runes[i] == '\n' {
					splitIdx = i + 1
					foundSplit = true
					break
				}
			}
		}

		// Priority 3: Sentence ending (.!?)
		if !foundSplit {
			for i := end; i > start+limit-searchLimit && i > start; i-- {
				c := runes[i]
				if (c == '。' || c == '！' || c == '？') && (i+1 < length && runes[i+1] == ' ') {
					splitIdx = i + 1
					foundSplit = true
					break
				}
			}
		}

		// Fallback: Hard limit
		if !foundSplit {
			splitIdx = end
		}

		chunks = append(chunks, string(runes[start:splitIdx]))
		start = splitIdx
	}

	return chunks
}
func boolPtr(b bool) *bool {
	return &b
}

// SplitTextForTTS splits a string into chunks small enough for TTS.
// It prioritizes splitting by sentence endings (。！？) and tries to keep chunks under maxChars.
func SplitTextForTTS(text string, maxChars int) []string {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return []string{text}
	}

	var chunks []string
	start := 0
	length := len(runes)

	for start < length {
		// If remaining is small enough, just add it
		if length-start <= maxChars {
			chunks = append(chunks, string(runes[start:]))
			break
		}

		// Find best split point
		// We want to split at maxChars, but we look back to find a valid punctuation
		targetEnd := start + maxChars
		if targetEnd > length {
			targetEnd = length
		}

		splitIdx := -1

		// Priority 1: Chinese Sentence Endings (。！？)
		// Scan backwards from targetEnd to start
		for i := targetEnd - 1; i > start; i-- {
			c := runes[i]
			// Check for sentence delimiters.
			// We split AFTER the delimiter so it stays with the previous sentence.
			if c == '。' || c == '！' || c == '？' || c == '；' {
				splitIdx = i + 1
				break
			}
		}

		// Priority 2: Comma (，) if no sentence ending found
		if splitIdx == -1 {
			for i := targetEnd - 1; i > start; i-- {
				if runes[i] == '，' {
					splitIdx = i + 1
					break
				}
			}
		}

		// Priority 3: Hard split if no punctuation found (rare but possible)
		if splitIdx == -1 {
			splitIdx = targetEnd
		}

		// Ensure we make progress
		if splitIdx <= start {
			splitIdx = start + 1
		}

		chunk := string(runes[start:splitIdx])
		chunks = append(chunks, chunk)
		start = splitIdx
	}

	return chunks
}

// GenerateAllAudio generates audio for all chapters sequentially, skipping chapters that already have audio files.
func GenerateAllAudio(c *gin.Context) {
	// Get all chapters
	chapters, ok := LoadedChapters["current"]
	if !ok || len(chapters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No book loaded"})
		return
	}

	Store.Mu.RLock()
	bookID := Store.BookID
	Store.Mu.RUnlock()

	// Start async batch generation
	go func() {
		total := len(chapters)
		successCount := 0
		skippedCount := 0
		failedChapters := []string{}

		cfg := config.Get()
		ttsClient := tts.NewClient(cfg)

		for i, chapter := range chapters {
			chapterID := chapter.ID
			chapterTitle := chapter.Title
			if chapterTitle == "" {
				chapterTitle = fmt.Sprintf("Chapter %s", chapterID)
			}

			// Broadcast overall progress
			overallPercent := int((float64(i) / float64(total)) * 100)
			BroadcastProgress("batch-generate", overallPercent, fmt.Sprintf("Processing %s (%d/%d)...", chapterTitle, i+1, total))

			// Check if audio already exists
			outDir := fmt.Sprintf("data/out/%s", bookID)
			audioPath := fmt.Sprintf("%s/%s.wav", outDir, chapterID)
			if _, err := os.Stat(audioPath); err == nil {
				log.Printf("[GenerateAll] Audio already exists for chapter %s, skipping", chapterID)
				skippedCount++
				continue
			}

			// Check if chapter has analysis data
			Store.Mu.RLock()
			segments, hasAnalysis := Store.Analysis[chapterID]
			Store.Mu.RUnlock()

			if !hasAnalysis || len(segments) == 0 {
				log.Printf("[GenerateAll] Chapter %s has no analysis data, skipping", chapterID)
				failedChapters = append(failedChapters, chapterTitle+" (no analysis)")
				continue
			}

			// Generate audio for this chapter (inline logic from GenerateAudio)
			tempDir := fmt.Sprintf("data/temp/%s", chapterID)
			os.MkdirAll(tempDir, 0755)
			os.MkdirAll(outDir, 0755)

			segTotal := len(segments)
			var filePaths []string
			chapterFailed := false

			for j, seg := range segments {
				Store.Mu.RLock()
				mapping, hasMapping := Store.VoiceMapping[seg.Speaker]
				Store.Mu.RUnlock()

				voice := ""
				emotion := seg.Emotion

				if hasMapping {
					voice = mapping.VoiceID

					useLLM := true
					if mapping.UseLLMEmotion != nil {
						useLLM = *mapping.UseLLMEmotion
					}

					if useLLM {
						if seg.Emotion != "" {
							emotion = seg.Emotion
						}
					} else {
						if mapping.Emotion != "" {
							emotion = mapping.Emotion
						}
					}
				}

				if emotion == "" {
					emotion = "calm"
				}

				// Progress for this segment within the chapter
				segPercent := int((float64(j) / float64(segTotal)) * 100)
				runes := []rune(seg.Text)
				display := string(runes)
				if len(runes) > 10 {
					display = string(runes[:10])
				}
				BroadcastProgress("batch-generate", overallPercent, fmt.Sprintf("%s (%d/%d) Seg %d/%d: %s...", chapterTitle, i+1, total, j+1, segTotal, display))

				textToSpeak := seg.Typesetting
				if textToSpeak == "" {
					textToSpeak = seg.Text
				}

				maxChars := 100
				chunks := SplitTextForTTS(textToSpeak, maxChars)

				var segmentAudioData []byte

				if len(chunks) == 1 {
					log.Printf("[GenerateAll] Chapter %s segment %d: %s", chapterID, j, textToSpeak)
					audioData, err := ttsClient.Generate(textToSpeak, voice, emotion)
					if err != nil {
						log.Printf("[GenerateAll] Error generating chapter %s segment %d: %v", chapterID, j, err)
						failedChapters = append(failedChapters, chapterTitle)
						chapterFailed = true
						break
					}
					segmentAudioData = audioData
				} else {
					log.Printf("[GenerateAll] Chapter %s segment %d is long, split into %d chunks", chapterID, j, len(chunks))
					var chunkFiles []string

					for k, chunk := range chunks {
						BroadcastProgress("batch-generate", overallPercent+segPercent/total, fmt.Sprintf("%s Seg %d/%d Part %d/%d", chapterTitle, j+1, segTotal, k+1, len(chunks)))
						chunkAudio, err := ttsClient.Generate(chunk, voice, emotion)
						if err != nil {
							log.Printf("[GenerateAll] Error chapter %s seg %d chunk %d: %v", chapterID, j, k, err)
							failedChapters = append(failedChapters, chapterTitle)
							chapterFailed = true
							break
						}

						chunkPath := fmt.Sprintf("%s/%d_part_%d.wav", tempDir, j, k)
						if err := os.WriteFile(chunkPath, chunkAudio, 0644); err != nil {
							log.Printf("[GenerateAll] Failed to write chunk: %v", err)
							chapterFailed = true
							break
						}
						chunkFiles = append(chunkFiles, chunkPath)
					}

					if chapterFailed {
						break
					}

					mergedPath := fmt.Sprintf("%s/%d_merged.wav", tempDir, j)
					if err := audio.MergeAndNormalize(chunkFiles, mergedPath, 0, false); err != nil {
						log.Printf("[GenerateAll] Failed to merge chunks: %v", err)
						failedChapters = append(failedChapters, chapterTitle)
						chapterFailed = true
						break
					}

					data, err := os.ReadFile(mergedPath)
					if err != nil {
						log.Printf("[GenerateAll] Failed to read merged: %v", err)
						chapterFailed = true
						break
					}
					segmentAudioData = data

					for _, f := range chunkFiles {
						os.Remove(f)
					}
					os.Remove(mergedPath)
				}

				if chapterFailed {
					break
				}

				filePath := fmt.Sprintf("%s/%d.wav", tempDir, j)
				if err := os.WriteFile(filePath, segmentAudioData, 0644); err != nil {
					log.Printf("[GenerateAll] Failed to write temp: %v", err)
					chapterFailed = true
					break
				}
				filePaths = append(filePaths, filePath)
			}

			if chapterFailed {
				os.RemoveAll(tempDir)
				continue
			}

			// Merge all segments for this chapter
			outPath := fmt.Sprintf("%s/%s.wav", outDir, chapterID)
			if err := audio.MergeAndNormalize(filePaths, outPath, cfg.MergeSilence, cfg.NormalizeAudio); err != nil {
				log.Printf("[GenerateAll] Merge failed for chapter %s: %v", chapterID, err)
				failedChapters = append(failedChapters, chapterTitle)
				os.RemoveAll(tempDir)
				continue
			}

			os.RemoveAll(tempDir)
			successCount++
			log.Printf("[GenerateAll] Chapter %s completed successfully", chapterID)
		}

		// Broadcast completion
		if len(failedChapters) > 0 {
			BroadcastProgress("batch-generate", 100, fmt.Sprintf("Done. %d generated, %d skipped, %d failed: %v", successCount, skippedCount, len(failedChapters), failedChapters))
		} else {
			BroadcastProgress("batch-generate", 100, fmt.Sprintf("All done! %d generated, %d skipped.", successCount, skippedCount))
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "started", "totalChapters": len(chapters)})
}
