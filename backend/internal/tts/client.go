package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"tts-book/backend/internal/config"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	client *resty.Client
	url    string
	cfg    *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: resty.New(),
		url:    cfg.IndexTTSUrl,
		cfg:    cfg,
	}
}

// Generate calls the Index-TTS API using the Gradio Event Protocol.
func (c *Client) Generate(text, voice, emotion string, speed float64) ([]byte, error) {
	// 1. Prepare Arguments
	if voice == "" {
		// Try to find a default voice in the configured voice directory
		voiceDir := c.cfg.VoiceDir
		if voiceDir == "" {
			voiceDir = "voices"
		}
		// Construct the path to a default voice, e.g., voice_06.wav if it exists, or just any wav
		defaultVoice := "voice_06.wav" // Or pick the first available
		voice = fmt.Sprintf("%s/%s", voiceDir, defaultVoice)
		// Check if it exists? We let the upload logic handle it or fail gracefully.
	}

	// 1.5 Upload voice if it's a local file
	if _, err := os.Stat(voice); err == nil {
		fmt.Printf("[TTS] Voice file found locally: %s. Uploading...\n", voice)
		remotePath, err := c.uploadVoice(voice)
		if err != nil {
			return nil, fmt.Errorf("failed to upload voice file: %v", err)
		}
		fmt.Printf("[TTS] Uploaded voice. Remote path: %s\n", remotePath)
		voice = remotePath
	} else {
		fmt.Printf("[TTS] Voice file not found locally or error: %v. Using as is: %s\n", err, voice)
	}

	// Emotion mapping to vectors
	emotions := map[string]int{
		"happy": 0, "angry": 1, "sad": 2, "afraid": 3,
		"disgusted": 4, "melancholic": 5, "surprised": 6, "calm": 7,
	}
	vecs := make([]float64, 8)
	if idx, ok := emotions[emotion]; ok {
		vecs[idx] = 1.0
	} else {
		vecs[7] = 1.0 // Default Calm
	}

	// Create FileData object for voice and emo_ref
	fileObj := map[string]interface{}{
		"path": voice,
		"meta": map[string]string{"_type": "gradio.FileData"},
	}

	// Construct data array (24 arguments)
	data := []interface{}{
		"Same as the voice reference",      // [0]
		fileObj,                            // [1] prompt (FileData object)
		text,                               // [2] text
		fileObj,                            // [3] emo_ref_path (Using same as voice)
		0.7,                                // [4] emo_weight
		vecs[0], vecs[1], vecs[2], vecs[3], // [5-8]
		vecs[4], vecs[5], vecs[6], vecs[7], // [9-12]
		"",    // [13] emo_text
		false, // [14] emo_random
		400,   // [15] max_text_tokens
		true,  // [16] do_sample
		0.8,   // [17] top_p
		30,    // [18] top_k
		0.8,   // [19] temperature
		0,     // [20] length_penalty
		3,     // [21] num_beams
		10,    // [22] repetition_penalty
		1500,  // [23] max_mel_tokens
	}

	payload := map[string]interface{}{"data": data}

	// Step 1: POST to get Event ID
	apiURL := fmt.Sprintf("%s/gradio_api/call/gen_single", c.url)
	var initResult struct {
		EventID string `json:"event_id"`
	}

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&initResult).
		Post(apiURL)

	if err != nil {
		return nil, fmt.Errorf("POST failed: %v", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("POST API error: %s - Body: %s", resp.Status(), resp.String())
	}
	if initResult.EventID == "" {
		return nil, fmt.Errorf("no event_id returned")
	}

	// Step 2: GET to read stream
	eventURL := fmt.Sprintf("%s/gradio_api/call/gen_single/%s", c.url, initResult.EventID)
	streamResp, err := c.client.R().SetDoNotParseResponse(true).Get(eventURL)
	if err != nil {
		return nil, fmt.Errorf("GET stream failed: %v", err)
	}
	defer streamResp.RawBody().Close()

	bodyBytes, _ := io.ReadAll(streamResp.RawBody())
	bodyString := string(bodyBytes)

	// Gradio Event Protocol sends several "event: ..." and "data: ..." blocks.
	// We are looking for "event: complete" or the last "data: " line.
	lines := strings.Split(bodyString, "\n")
	var lastDataLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			lastDataLine = strings.TrimPrefix(line, "data: ")
		}
	}

	if lastDataLine == "" {
		return nil, fmt.Errorf("no data found in stream response")
	}

	// Parse the data list (it's a list containing the file object)
	var dataList []interface{}
	if err := json.Unmarshal([]byte(lastDataLine), &dataList); err != nil {
		return nil, fmt.Errorf("failed to parse result data: %v", err)
	}

	if len(dataList) == 0 {
		return nil, fmt.Errorf("empty result data")
	}

	// Gradio returns a file object in dataList[0]
	// { "path": "...", "url": "...", "orig_name": "...", ... }
	var resultFile string
	if fileObj, ok := dataList[0].(map[string]interface{}); ok {
		// Handle Gradio update wrapper { "__type__": "update", "value": { ... } }
		if val, ok := fileObj["value"].(map[string]interface{}); ok {
			fileObj = val
		}

		if path, ok := fileObj["url"].(string); ok {
			resultFile = path
		} else if path, ok := fileObj["path"].(string); ok {
			resultFile = path
		}
	} else if path, ok := dataList[0].(string); ok {
		resultFile = path
	}

	if resultFile == "" {
		return nil, fmt.Errorf("could not find audio path in result: %v", dataList[0])
	}

	// Download the audio
	fileURL := resultFile
	if !strings.HasPrefix(resultFile, "http") {
		fileURL = fmt.Sprintf("%s/file=%s", c.url, resultFile)
	}

	audioResp, err := c.client.R().Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio from %s: %v", fileURL, err)
	}
	if audioResp.IsError() {
		return nil, fmt.Errorf("failed to download audio error: %s", audioResp.Status())
	}

	audioData := audioResp.Body()

	// Apply speed adjustment if needed
	if speed != 1.0 {
		processedAudio, err := c.applySpeedAdjustment(audioData, speed)
		if err != nil {
			// Log the error but return original audio instead of failing completely
			fmt.Printf("[TTS] Warning: Failed to apply speed adjustment (%.2fx): %v\n", speed, err)
			fmt.Printf("[TTS] Returning original audio without speed adjustment\n")
			return audioData, nil
		}
		return processedAudio, nil
	}

	return audioData, nil
}

// applySpeedAdjustment uses FFmpeg to adjust audio playback speed without changing pitch
func (c *Client) applySpeedAdjustment(audioData []byte, speed float64) ([]byte, error) {
	// Create temporary files for input and output
	tmpInput, err := os.CreateTemp("", "tts-input-*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %v", err)
	}
	defer os.Remove(tmpInput.Name())
	defer tmpInput.Close()

	tmpOutput, err := os.CreateTemp("", "tts-output-*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %v", err)
	}
	defer os.Remove(tmpOutput.Name())
	defer tmpOutput.Close()

	// Write audio data to temp input file
	if _, err := tmpInput.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %v", err)
	}
	tmpInput.Close()

	// Build FFmpeg atempo filter chain
	// atempo filter range is 0.5 to 2.0, so we may need to chain multiple filters
	var atempoFilters []string
	remainingSpeed := speed

	for remainingSpeed > 0 {
		if remainingSpeed >= 2.0 {
			atempoFilters = append(atempoFilters, "atempo=2.0")
			remainingSpeed /= 2.0
		} else if remainingSpeed <= 0.5 {
			atempoFilters = append(atempoFilters, "atempo=0.5")
			remainingSpeed /= 0.5
		} else {
			atempoFilters = append(atempoFilters, fmt.Sprintf("atempo=%.4f", remainingSpeed))
			break
		}
	}

	filterChain := strings.Join(atempoFilters, ",")

	// Build FFmpeg command
	// ffmpeg -i input.wav -af "atempo=X.X" output.wav
	cmdArgs := []string{
		"-i", tmpInput.Name(),
		"-af", filterChain,
		"-y", // Overwrite output file
		tmpOutput.Name(),
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %v, stderr: %s", err, stderr.String())
	}

	// Read the processed audio
	processedData, err := os.ReadFile(tmpOutput.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read processed audio: %v", err)
	}

	fmt.Printf("[TTS] Successfully applied speed adjustment: %.2fx (filters: %s)\n", speed, filterChain)
	return processedData, nil
}

func (c *Client) uploadVoice(filePath string) (string, error) {
	uploadURL := fmt.Sprintf("%s/gradio_api/upload", c.url)

	resp, err := c.client.R().
		SetFile("files", filePath).
		Post(uploadURL)

	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("upload failed: %s", resp.Status())
	}

	// Response is usually ["/tmp/gradio/...", ...]
	var paths []string
	if err := json.Unmarshal(resp.Body(), &paths); err != nil {
		return "", err
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("no paths returned from upload")
	}
	return paths[0], nil
}
