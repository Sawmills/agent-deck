package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements AIProvider for OpenAI's API
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider instance
// baseURL is optional and defaults to OpenAI's endpoint
func NewOpenAIProvider(apiKey, model string, baseURL ...string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("model name is required")
	}

	config := openai.DefaultConfig(apiKey)

	// Set custom base URL if provided (e.g., for OpenRouter)
	if len(baseURL) > 0 && baseURL[0] != "" {
		config.BaseURL = baseURL[0]
	}

	client := openai.NewClientWithConfig(config)

	return &OpenAIProvider{
		client: client,
		model:  model,
	}, nil
}

// Chat sends messages to OpenAI and returns a single response
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	// Convert our Message format to OpenAI's format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "user":
			openaiMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Content,
			}
		case "assistant":
			openaiMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: msg.Content,
			}
		default:
			return "", fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	// Call the OpenAI API
	response, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return "", fmt.Errorf("openai API error: %w", err)
	}

	// Extract the text response
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI API")
	}

	return response.Choices[0].Message.Content, nil
}

// ChatStream sends messages to OpenAI and returns a channel of response chunks
// Currently returns an error as streaming is not yet implemented
func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
	return nil, fmt.Errorf("streaming not implemented yet")
}
