package ai

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements AIProvider for Anthropic's Claude API
type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(apiKey, model string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("model name is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &AnthropicProvider{
		client: client,
		model:  model,
	}, nil
}

// Chat sends messages to Claude and returns a single response
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (response string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in Anthropic provider: %v", r)
		}
	}()
	// Convert our Message format to Anthropic's format
	anthropicMessages := make([]anthropic.MessageParam, len(messages))
	for i, msg := range messages {
		// Create a text block for the content
		textBlock := anthropic.NewTextBlock(msg.Content)

		// Create the message with the appropriate role
		if msg.Role == "user" {
			anthropicMessages[i] = anthropic.NewUserMessage(textBlock)
		} else if msg.Role == "assistant" {
			anthropicMessages[i] = anthropic.NewAssistantMessage(textBlock)
		} else {
			return "", fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	// Call the Anthropic API
	apiResponse, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMessages,
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract the text response
	if len(apiResponse.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic API")
	}

	// The first content block should be text
	firstBlock := apiResponse.Content[0]
	if firstBlock.Type != "text" {
		return "", fmt.Errorf("unexpected response type from Anthropic API: %s", firstBlock.Type)
	}

	response = firstBlock.Text
	return response, nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
	anthropicMessages := make([]anthropic.MessageParam, len(messages))
	for i, msg := range messages {
		textBlock := anthropic.NewTextBlock(msg.Content)
		if msg.Role == "user" {
			anthropicMessages[i] = anthropic.NewUserMessage(textBlock)
		} else if msg.Role == "assistant" {
			anthropicMessages[i] = anthropic.NewAssistantMessage(textBlock)
		} else {
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  anthropicMessages,
	})

	chunks := make(chan string, 100)

	go func() {
		defer close(chunks)

		for stream.Next() {
			event := stream.Current()
			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					select {
					case chunks <- deltaVariant.Text:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		if stream.Err() != nil {
			select {
			case chunks <- fmt.Sprintf("\n\n[Error: %v]", stream.Err()):
			case <-ctx.Done():
			}
		}
	}()

	return chunks, nil
}
