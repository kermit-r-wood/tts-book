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

关键规则 1：分割对话与旁白
必须将“角色对话”与“旁白描述”彻底分开。
即使它们出现在同一段落或同一行中，也必须拆分为不同的片段。
- 旁白 (Narration)：描述动作、场景或心理活动的文本（如：他说、她想）。Speaker 必须设为 'Narrator'。
- 对话 (Dialogue)：角色说的话，通常包含在引号中（如 “...” 或 "..."）。Speaker 必须设为角色名称。

关键规则 2：严格处理同段落混合文本
当一段文本中同时包含“旁白前的动作”、“对话”和“对话后的动作”时，必须将其切分为三个独立的 JSON 对象。
绝对禁止将对话引号外的任何文字（特别是对话后的动作描述）合并到对话内容的 Speaker 中。

示例错误处理：
错误：{"text": "“你不该挑起这副重担。”说完他就死了。", "speaker": "角色名"}
正确：
[
  {"text": "“你不该挑起这副重担。”", "speaker": "角色名"},
  {"text": "说完他就死了。", "speaker": "Narrator"}
]

关键规则 3：多音字拼音标注
为了修正 TTS (语音合成) 的多音字发音错误，请检测文本中【易读错】的多音字，并将其【替换】为“对应拼音+声调”的格式。
Index-TTS 支持混合字音输入。
格式：[大写拼音][数字声调]
注意：请使用 checkpoints/pinyin.vocab 中支持的拼音组合。
常见易错示例：
- "的": 
  - 结构助词 -> DE5 (如 '漂亮的' -> '漂亮DE5')
  - '打的' (Taxi) -> '打DI1'
  - '的确' -> 'DI2确'
- "得":
  - 结构助词 (verb+得+adj) -> DE5 (如 '跑得快' -> '跑DE5快')
  - '不得不' -> '不DEI3不'
- "地":
  - 结构助词 (adj+地+verb) -> DE5 (如 '高兴地' -> '高兴DE5')
- "行":
  - '不行' -> '不XING2'
  - '行业' -> 'HANG2业'
- "着":
  - '看着' -> '看ZHE5'
  - '着火' -> 'ZHAO2火'
- "都":
  - '都是' -> 'DU1是'
  - '都市' -> 'DU1市'

关键规则 4：精准识别角色名称 (Contextual Speaker Inference)
当 Speaker 不明确时，必须根据上下文（Context）推理出具体的角色名称或身份，【严禁】使用“男角色”、“女角色”、“某人”等泛指。

推理优先级：
1. 明确的说话引导语：如 "xxx说"、"xxx道"。
2. 紧邻的动作执行者：如果对话前没有 "说"，则取上一句动作的主语。
   例如：“父亲抓住她的手腕……‘对话’” -> Speaker 为 “父亲”。
3. 上文提及的全名：如果角色在上文中被介绍过（如 “加伯·蒙洛卡托”），且当前被称为 “父亲”，优先使用最具体的名字（本例中 “父亲” 或 “加伯·蒙洛卡托” 均优于 “男角色”）。

要求：
请根据上下文语境，判断多音字的正确读音。
如果该多音字在 TTS 中容易混淆，请务必将其替换为拼音。
对于几乎不会读错的固定词组（如“银行”），可以保留汉字。

综合示例：
输入：
蒙扎十四岁那年，加伯·蒙洛卡托发起高烧。她和本纳眼睁睁看着他咳嗽。
某天夜里，父亲抓住她的手腕，盯着她。
“明天必须给上面的田地松土，尽可能多种些东西。”

输出：
{"segments": [
  {"text": "蒙扎十四岁那年，加伯·蒙洛卡托发起高烧。她和本纳眼睁睁看着他咳嗽。", "speaker": "Narrator", "emotion": "sad"},
  {"text": "某天夜里，父亲抓住她的手腕，盯着她。", "speaker": "Narrator", "emotion": "intense"},
  {"text": "“明天必须给上面的田地松土，尽可能多种些东西。”", "speaker": "加伯·蒙洛卡托", "emotion": "calm"}
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
