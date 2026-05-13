package client

import (
	"context"
	"fmt"
	"github.com/akikohatsune/teto/config"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type ImageInput struct {
	MimeType string `json:"mime_type"`
	DataB64  string `json:"data_b64"`
}

type ChatMessage struct {
	Role    string       `json:"role"`
	Content string       `json:"content"`
	Images  []ImageInput `json:"images,omitempty"`
}

type LLMClient struct {
	settings *config.Settings
	client   *openai.Client
}

func NewLLMClient(settings *config.Settings) *LLMClient {
	config := openai.DefaultConfig(settings.NvidiaAPIKey)
	config.BaseURL = "https://integrate.api.nvidia.com/v1"
	
	return &LLMClient{
		settings: settings,
		client:   openai.NewClientWithConfig(config),
	}
}

func (c *LLMClient) Generate(ctx context.Context, messages []ChatMessage) (string, error) {
	openaiMessages := c.buildOpenAIMessages(messages)

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.settings.NvidiaModel,
			Messages:    openaiMessages,
			Temperature: float32(c.settings.Temperature),
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		content := resp.Choices[0].Message.Content
		if strings.TrimSpace(content) != "" {
			return strings.TrimSpace(content), nil
		}
	}

	return "", fmt.Errorf("NVIDIA returned an empty response")
}

func (c *LLMClient) buildOpenAIMessages(messages []ChatMessage) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage

	// Add system prompt
	result = append(result, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: c.settings.SystemPrompt,
	})

	for _, msg := range messages {
		if len(msg.Images) > 0 {
			var multiContent []openai.ChatMessagePart
			if msg.Content != "" {
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: msg.Content,
				})
			}
			for _, img := range msg.Images {
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL:    fmt.Sprintf("data:%s;base64,%s", img.MimeType, img.DataB64),
						Detail: openai.ImageURLDetailHigh,
					},
				})
			}
			result = append(result, openai.ChatCompletionMessage{
				Role:         msg.Role,
				MultiContent: multiContent,
			})
		} else {
			result = append(result, openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return result
}


