package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"sync"
	"time"

	"tts-book/backend/internal/config"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

const systemPrompt = `
	分析提供的文本，并严格将其分割为 JSON 对象列表。
	必须包含输入中的【所有文本】，并保持【原始顺序】。
	返回必须是一个 JSON 对象，包含 "segments" 数组。

	关键规则 1：【强制】分割对话与旁白 (MANDATORY Split)
	核心原则：**严禁**在一个 JSON 对象中同时包含引号内的内容（对话）和引号外的内容（旁白）。
	
	执行步骤：
	1. 扫描文本，找到所有的引号（“...” 或 "..."）。
	2. 将引号内的部分提取为 { "speaker": "角色名", ... }。
	3. 将引号外的部分（包括描述、动作、标点）提取为 { "speaker": "Narrator", ... }。
	4. 保持原文的物理顺序。

	常见结构处理：
	1. [动作, 对话]：
	   原文：他摸了摸她的脸，“你不该挑起这副重担，但你弟弟太小。”
	   拆分：
	   - {"text": "他摸了摸她的脸，", "speaker": "Narrator", ...}
	   - {"text": "“你不该挑起这副重担，但你弟弟太小。”", "speaker": "加伯·蒙洛卡托", ...}
	   注意：【，】归属旁白。

	2. [对话, 动作]：
	   原文：“快跑！”他大喊。
	   拆分：
	   - {"text": "“快跑！”", "speaker": "角色名", ...}
	   - {"text": "他大喊。", "speaker": "Narrator", ...}

	3. [对话, 动作, 对话] (三明治结构)：
	   原文：“蒙扎，”他会笑眯眯地俯视她，“没有你我该怎么办？”
	   拆分：
	   - {"text": "“蒙扎，”", "speaker": "加伯", ...}
	   - {"text": "他会笑眯眯地俯视她，", "speaker": "Narrator", ...}
	   - {"text": "“没有你我该怎么办？”", "speaker": "加伯", ...}

	4. [复杂交替] (Complex Interleaved):
	   原文：她叹了口气，“事实就是事实。”她在马鞍上伸个懒腰，“不过，我爱听。”
	   拆分：
	   - {"text": "她叹了口气，", "speaker": "Narrator"} (动作指示主体)
	   - {"text": "“事实就是事实。”", "speaker": "她(角色名)", "emotion": "melancholic"} (由叹气推断)
	   - {"text": "她在马鞍上伸个懒腰，", "speaker": "Narrator"}
	   - {"text": "“不过，我爱听。”", "speaker": "她(角色名)", "emotion": "calm"} (由伸懒腰恢复平静)

	- 旁白 (Narrator)：描述动作、场景。Speaker: 'Narrator'。
	- 对话 (Dialogue)：引号内的内容。Speaker: 角色名称。

	关键规则 2：Typesetting 字段 (Pinyin Annotation)
	"typesetting" 字段用于语音合成。
	1. 【默认行为】：完全复制 "text" 字段的内容。
	2. 【仅修改多音字】：只有在遇到以下列表中的多音字时，才将其替换为【对应的大写拼音+声调数字】。
	   【重要】：仅替换多音字字符本身，**严禁**吞掉后面的词。
	   正确示例： "难产" -> "NAN2产"
	   错误示例： "难产" -> "NAN2"
	3. 【严禁】：不要替换非多音字。不要留空。

	多音字强制替换列表 (Mandatory Pinyin Replacement):
	- 【行】：XH2 (银行) / XING2 (行为)
	- 【得】：DEI3 (得去) / DE2 (跑得快) / DE5 (觉得)
	- 【地】：DI4 (田地) / DE5 (慢慢地)
	- 【重】：CHONG2 (重新) / ZHONG4 (重要, 重担)
	- 【着】：ZHAO2 (着火) / ZHE5 (看着) / ZHUO2 (着装)
	- 【长】：CHANG2 (长短) / ZHANG3 (长大)
	- 【乐】：LE4 (快乐) / YUE4 (音乐)
	- 【好】：HAO3 (好人) / HAO4 (爱好)
	- 【干】：GAN1 (干净) / GAN4 (干活)
	- 【难】：NAN2 (难产, 困难, 为难) / NAN4 (灾难, 难民)

	关键规则 3：精准识别角色 (Contextual Speaker Inference)
	根据上下文推理角色名称，严禁使用“男角色”、“女角色”。

	示例：
	输入：
	他摸了摸她的脸，“你不该挑起这副重担，但你弟弟太小。”

	输出：
	{
	  "segments": [
		{
		  "text": "他摸了摸她的脸，",
		  "typesetting": "他摸了摸她的脸，",
		  "speaker": "Narrator",
		  "emotion": "calm"
		},
		{
		  "text": "“你不该挑起这副重担，但你弟弟太小。”",
		  "typesetting": "“你不该挑起这副ZHONG4担，但你弟弟太小。”",
		  "speaker": "加伯·蒙洛卡托",
		  "emotion": "sad"
		}
	  ]
	}

	Emotion 必须是以下之一：[happy, angry, sad, afraid, disgusted, melancholic, surprised, calm]。
	默认为 'calm'。
	仅返回严格有效的 JSON。`

type AnalysisResult struct {
	Text        string `json:"text"`
	Typesetting string `json:"typesetting,omitempty"` // Text with Pinyin annotations for TTS
	Speaker     string `json:"speaker"`
	Emotion     string `json:"emotion"`
}

type Client struct {
	api             *openai.Client
	genaiClient     *genai.Client
	mu              sync.Mutex
	lastRequestTime time.Time
	minInterval     time.Duration
	isMock          bool
	model           string
	provider        string
	apiKey          string
}

func NewClient(cfg *config.Config) *Client {
	c := openai.DefaultConfig(cfg.LLMAPIKey)
	c.BaseURL = cfg.LLMBaseURL

	client := &Client{
		api:         openai.NewClientWithConfig(c),
		minInterval: time.Duration(cfg.LLMMinInterval) * time.Millisecond,
		isMock:      cfg.MockLLM,
		provider:    cfg.LLMProvider,
		apiKey:      cfg.LLMAPIKey,
	}

	if cfg.LLMProvider == "gemini" {
		ctx := context.Background()
		gClient, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  cfg.LLMAPIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			log.Printf("[LLM] Failed to create Gemini client: %v", err)
		} else {
			client.genaiClient = gClient
		}
	}

	return client
}

func (c *Client) AnalyzeTextStream(text string, onToken func(string)) ([]AnalysisResult, error) {
	if c.isMock {
		log.Println("[LLM] Mock Mode Enabled. Returning simulated result.")
		time.Sleep(1 * time.Second)

		// Simulate tokens
		mockTokens := []string{"{", "\"segments\": ", "[", "{", "\"text\": ", "\"Mock segment 1\"", ", \"speaker\": ", "\"Narrator\"", ", \"emotion\": ", "\"calm\"", "},", "{", "\"text\": ", "\"Mock dialogue\"", ", \"speaker\": ", "\"Hero\"", ", \"emotion\": ", "\"happy\"", "}]", "}"}
		for _, t := range mockTokens {
			if onToken != nil {
				onToken(t)
			}
			time.Sleep(50 * time.Millisecond)
		}

		return []AnalysisResult{
			{Text: "This is a mock narration segment.", Speaker: "Narrator", Emotion: "calm"},
			{Text: "This is a mock dialogue segment.", Speaker: "Hero", Emotion: "happy"},
			{Text: "Another mock narration.", Speaker: "Narrator", Emotion: "calm"},
		}, nil
	}

	if c.provider == "gemini" {
		return c.streamGemini(text, onToken)
	}

	// Rate Limiting: Cooldown
	c.mu.Lock()
	elapsed := time.Since(c.lastRequestTime)
	if elapsed < c.minInterval {
		wait := c.minInterval - elapsed
		log.Printf("[LLM] Rate Limit Cooldown: Waiting %v...\n", wait)
		time.Sleep(wait)
	}
	c.mu.Unlock()

	// Update lastRequestTime when we return
	defer func() {
		c.mu.Lock()
		c.lastRequestTime = time.Now()
		c.mu.Unlock()
	}()

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[LLM] Sending Streaming request (Length: %d, Attempt: %d/%d)\n", len(text), attempt, maxRetries)

		var messages []openai.ChatCompletionMessage
		messages = []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: systemPrompt + "\n\n" + text},
		}

		req := openai.ChatCompletionRequest{
			Model:    c.model,
			Messages: messages,
			Stream:   true,
		}

		stream, err := c.api.CreateChatCompletionStream(context.Background(), req)
		if err != nil {
			var apiErr *openai.APIError
			if errors.As(err, &apiErr) {
				log.Printf("[LLM] Stream Creation API Error (Attempt %d): StatusCode=%d, Code=%s, Message=%s\n", attempt, apiErr.HTTPStatusCode, apiErr.Code, apiErr.Message)
			} else {
				log.Printf("[LLM] Stream Creation Error (Attempt %d): %v\n", attempt, err)
			}

			lastErr = err
			time.Sleep(1 * time.Second) // Basic backoff
			continue
		}

		var fullContent strings.Builder
		streamFailed := false
		var finishReason string

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				var apiErr *openai.APIError
				if errors.As(err, &apiErr) {
					log.Printf("[LLM] Stream Recv API Error (Attempt %d): StatusCode=%d, Code=%s, Message=%s\n", attempt, apiErr.HTTPStatusCode, apiErr.Code, apiErr.Message)
				} else {
					log.Printf("[LLM] Stream Recv Error (Attempt %d): %v\n", attempt, err)
				}
				streamFailed = true
				break
			}

			// Check if Choices array is empty to prevent index out of range panic
			if len(response.Choices) == 0 {
				log.Printf("[LLM] Warning: Received response with empty Choices array (Attempt %d)\n", attempt)
				continue
			}

			if len(response.Choices) > 0 {
				finishReason = string(response.Choices[0].FinishReason)
			}

			token := response.Choices[0].Delta.Content
			if token != "" {
				fullContent.WriteString(token)
				if onToken != nil {
					onToken(token) // Note: This might send partial tokens from failed attempts to UI, which is acceptable for now
				}
			}
		}
		stream.Close()
		log.Printf("[LLM] Stream Finished (Attempt %d). FinishReason: %s\n", attempt, finishReason)

		if streamFailed {
			lastErr = fmt.Errorf("stream interrupted")
			time.Sleep(1 * time.Second)
			continue
		}

		content := fullContent.String()
		log.Printf("[LLM] Full Response Accumulated for Parsing (Attempt %d).\n", attempt)

		// Robust JSON Extraction
		jsonContent, err := extractJSON(content)
		if err != nil {
			log.Printf("[LLM] JSON Extraction Error (Attempt %d): %v. Content: %s\n", attempt, err, content)
			lastErr = fmt.Errorf("failed to extract JSON: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Helper struct for JSON Mode response wrapper
		type responseWrapper struct {
			Segments []AnalysisResult `json:"segments"`
		}

		var wrapper responseWrapper
		parseErr := json.Unmarshal([]byte(jsonContent), &wrapper)

		// If parse failed and we suspect truncation, try to repair *before* complaining
		if parseErr != nil && (finishReason == "content_filter" || finishReason == "length") {
			log.Printf("[LLM] Warning: Response truncated (FinishReason: %s). Attempting to repair JSON...\n", finishReason)

			// Simple repair: Try appending closing structures
			// The extractJSON likely gave us something starting with '{'
			// We tried to parse { "segments": [ ... ]
			// It might be cut off like { "segments": [ { ... }, { ...

			// strategy 1: Close array and object
			repairedContent := jsonContent + "]}"
			if errRepair := json.Unmarshal([]byte(repairedContent), &wrapper); errRepair == nil {
				log.Printf("[LLM] JSON repair successful. Proceeding with salvaged data.\n")
				return wrapper.Segments, nil
			}

			// strategy 2: maybe it was cut off inside a string or object?
			// tough to fix perfectly without a complex parser, but let's try a slightly more aggressive cut
			// Find last '},' and cut there, then close.
			if lastComma := strings.LastIndex(jsonContent, "},"); lastComma != -1 {
				repairedContent = jsonContent[:lastComma+1] + "]}"
				if errRepair := json.Unmarshal([]byte(repairedContent), &wrapper); errRepair == nil {
					log.Printf("[LLM] JSON repair successful (aggressive cut). Proceeding with salvaged data.\n")
					return wrapper.Segments, nil
				}
			}
		}

		if parseErr != nil {
			log.Printf("[LLM] JSON Parse Error (Attempt %d): %v. Content End: ...%s\n", attempt, parseErr, getLastNChars(jsonContent, 100))
			lastErr = fmt.Errorf("failed to parse LLM JSON: %v. Check log for content", parseErr)
			time.Sleep(1 * time.Second)
			continue
		}

		// Success!
		return wrapper.Segments, nil
	}

	return nil, fmt.Errorf("analysis failed after %d attempts. Last error: %v", maxRetries, lastErr)
}

func getLastNChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func (c *Client) ListModels() ([]string, error) {
	if c.isMock {
		return []string{"mock-model-1", "mock-model-2"}, nil
	}

	c.mu.Lock()
	if c.provider == "gemini" {
		c.mu.Unlock()
		return []string{"gemini-3-flash-preview", "gemini-2.0-flash-exp", "gemini-1.5-flash", "gemini-1.5-pro"}, nil
	}
	// Rate limit check could be here if needed
	c.mu.Unlock()

	list, err := c.api.ListModels(context.Background())
	if err != nil {
		return nil, err
	}

	var models []string
	for _, m := range list.Models {
		models = append(models, m.ID)
	}
	return models, nil
}

// streamGemini handles the native Google Gemini API streaming using GenAI SDK
func (c *Client) streamGemini(text string, onToken func(string)) ([]AnalysisResult, error) {
	if c.genaiClient == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	// Rate Limiting
	c.mu.Lock()
	elapsed := time.Since(c.lastRequestTime)
	if elapsed < c.minInterval {
		time.Sleep(c.minInterval - elapsed)
	}
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.lastRequestTime = time.Now()
		c.mu.Unlock()
	}()

	maxRetries := 3
	var lastErr error

	// Default to gemini-3-flash-preview if not specified
	model := c.model
	if model == "" {
		model = "gemini-3-flash-preview"
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[LLM] Sending Gemini SDK Streaming request (Length: %d, Attempt: %d/%d)\n", len(text), attempt, maxRetries)

		ctx := context.Background()

		// Use genai.Text helper to create contents
		contents := genai.Text(systemPrompt + "\n\n" + text)

		var fullContent strings.Builder
		streamFailed := false

		// Use range loop for iterator
		for resp, err := range c.genaiClient.Models.GenerateContentStream(ctx, model, contents, nil) {
			if err != nil {
				log.Printf("[LLM] SDK Stream Recv Error (Attempt %d): %v\n", attempt, err)
				streamFailed = true
				break
			}

			if resp != nil && len(resp.Candidates) > 0 {
				for _, part := range resp.Candidates[0].Content.Parts {
					// Using fmt.Sprint as genai.Text is a function, not a type we can assert to directly here without knowing the internal struct name.
					// The part is likely a struct that String()s nicely or we can refine later.
					// Access Text field directly. Assuming Part is a struct with Text field.
					txt := part.Text
					fullContent.WriteString(txt)
					if onToken != nil {
						onToken(txt)
					}
				}
			}
		}

		if streamFailed {
			lastErr = fmt.Errorf("stream interrupted")
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[LLM] Stream Finished (Attempt %d).\n", attempt)

		content := fullContent.String()
		log.Printf("[LLM] Full Response Accumulated for Parsing (Attempt %d).\n", attempt)

		// Robust JSON Extraction (Same as OpenAI)
		jsonContent, err := extractJSON(content)
		if err != nil {
			log.Printf("[LLM] JSON Extraction Error (Attempt %d): %v\n", attempt, err)
			lastErr = fmt.Errorf("failed to extract JSON: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		type responseWrapper struct {
			Segments []AnalysisResult `json:"segments"`
		}

		var wrapper responseWrapper
		parseErr := json.Unmarshal([]byte(jsonContent), &wrapper)

		if parseErr != nil {
			log.Printf("[LLM] JSON Parse Error (Attempt %d): %v.\n", attempt, parseErr)

			// Try repair (same as existing)
			repairedContent := jsonContent + "]}"
			if errRepair := json.Unmarshal([]byte(repairedContent), &wrapper); errRepair == nil {
				log.Printf("[LLM] JSON repair successful.\n")
				return wrapper.Segments, nil
			}

			lastErr = fmt.Errorf("failed to parse LLM JSON: %v", parseErr)
			time.Sleep(1 * time.Second)
			continue
		}

		return wrapper.Segments, nil
	}

	return nil, fmt.Errorf("analysis failed after %d attempts. Last error: %v", maxRetries, lastErr)
}

// extractJSON attempts to find the first '{' and last '}' to extract the valid JSON object
func extractJSON(s string) (string, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || start > end {
		return "", fmt.Errorf("no valid JSON object found in response")
	}

	return s[start : end+1], nil
}
