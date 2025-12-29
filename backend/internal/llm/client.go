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
)

type AnalysisResult struct {
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
	Emotion string `json:"emotion"`
}

type Client struct {
	api             *openai.Client
	mu              sync.Mutex
	lastRequestTime time.Time
	minInterval     time.Duration
	isMock          bool
	model           string
}

func NewClient(cfg *config.Config) *Client {
	c := openai.DefaultConfig(cfg.LLMAPIKey)
	c.BaseURL = cfg.LLMBaseURL
	return &Client{
		api:         openai.NewClientWithConfig(c),
		minInterval: time.Duration(cfg.LLMMinInterval) * time.Millisecond,
		isMock:      cfg.MockLLM,
		model:       cfg.LLMModel,
	}
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

	prompt := `分析提供的文本，并严格将其分割为 JSON 对象列表。
必须包含输入中的【所有文本】，并保持【原始顺序】。
不得跳过任何文本，也不得进行摘要。

关键规则：必须将“角色对话”与“旁白描述”彻底分开。
即使它们出现在同一段落或同一行中，也必须拆分为不同的片段。
- 旁白 (Narration)：描述动作、场景或心理活动的文本（如：他说、她想）。Speaker 必须设为 'Narrator'。
- 对话 (Dialogue)：角色说的话，通常包含在引号中（如 “...” 或 "..."）。Speaker 必须设为角色名称。

示例输入：
张三抬起头。“你好，”他轻声说。

示例输出：
{"segments": [
  {"text": "张三抬起头。", "speaker": "Narrator", "emotion": "calm"},
  {"text": "“你好，”", "speaker": "张三", "emotion": "calm"},
  {"text": "他轻声说。", "speaker": "Narrator", "emotion": "calm"}
]}

返回一个包含 "segments" 键的 JSON 对象：{"segments": [{"text": "...", "speaker": "...", "emotion": "..."}]}
重要："segments" 列表必须包含对象，而不是字符串。
Emotion 必须是以下之一：[happy, angry, sad, afraid, disgusted, melancholic, surprised, calm]。
默认为 'calm'。
仅返回严格有效的 JSON。`

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[LLM] Sending Streaming request (Length: %d, Attempt: %d/%d)\n", len(text), attempt, maxRetries)

		req := openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: prompt},
				{Role: openai.ChatMessageRoleUser, Content: text},
			},
			Stream: true,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		}

		stream, err := c.api.CreateChatCompletionStream(context.Background(), req)
		if err != nil {
			log.Printf("[LLM] Stream Creation Error (Attempt %d): %v\n", attempt, err)
			lastErr = err
			time.Sleep(1 * time.Second) // Basic backoff
			continue
		}

		var fullContent strings.Builder
		streamFailed := false

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("[LLM] Stream Recv Error (Attempt %d): %v\n", attempt, err)
				streamFailed = true
				break
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

		if streamFailed {
			lastErr = fmt.Errorf("stream interrupted")
			time.Sleep(1 * time.Second)
			continue
		}

		content := fullContent.String()
		log.Printf("[LLM] Full Response Accumulated for Parsing (Attempt %d).\n", attempt)

		// Helper struct for JSON Mode response wrapper
		type responseWrapper struct {
			Segments []AnalysisResult `json:"segments"`
		}

		var wrapper responseWrapper
		if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
			log.Printf("[LLM] JSON Parse Error (Attempt %d): %v. Content: %s\n", attempt, err, content)
			lastErr = fmt.Errorf("failed to parse LLM JSON: %v. Check log for content", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Success!
		return wrapper.Segments, nil
	}

	return nil, fmt.Errorf("analysis failed after %d attempts. Last error: %v", maxRetries, lastErr)
}

func (c *Client) ListModels() ([]string, error) {
	if c.isMock {
		return []string{"mock-model-1", "mock-model-2"}, nil
	}

	c.mu.Lock()
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
