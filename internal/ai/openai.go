package ai

import (
	"context"
	"errors"
	"fmt"
	"io"

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
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (response string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in OpenAI provider: %v", r)
		}
	}()
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
	apiResponse, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return "", fmt.Errorf("openai API error: %w", err)
	}

	// Extract the text response
	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI API")
	}

	response = apiResponse.Choices[0].Message.Content
	return response, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
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
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("openai streaming error: %w", err)
	}

	chunks := make(chan string, 100)

	go func() {
		defer close(chunks)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				select {
				case chunks <- fmt.Sprintf("\n\n[Error: %v]", err):
				case <-ctx.Done():
				}
				return
			}

			if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
				select {
				case chunks <- response.Choices[0].Delta.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return chunks, nil
}
