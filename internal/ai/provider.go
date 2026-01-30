package ai

import (
	"context"
	"fmt"
)

// Message represents a single message in a conversation
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// AIProvider is the interface for AI provider implementations
// Each AI provider (Claude, Gemini, OpenAI, etc.) implements this interface
type AIProvider interface {
	// Chat sends messages and returns a single response
	Chat(ctx context.Context, messages []Message) (string, error)
	// ChatStream sends messages and returns a channel of response chunks
	ChatStream(ctx context.Context, messages []Message) (<-chan string, error)
}

// NewProvider creates a new AI provider instance
// providerType: "claude", "gemini", "openai", "openrouter", etc.
// apiKey: API key for the provider
// model: model name/ID for the provider
func NewProvider(providerType, apiKey, model string) (AIProvider, error) {
	switch providerType {
	case "anthropic":
		return NewAnthropicProvider(apiKey, model)
	case "openai":
		return NewOpenAIProvider(apiKey, model)
	case "openrouter":
		return NewOpenAIProvider(apiKey, model, "https://openrouter.ai/api/v1")
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
