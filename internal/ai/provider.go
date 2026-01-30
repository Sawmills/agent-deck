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
// providerType: "claude", "gemini", "openai", etc.
// apiKey: API key for the provider
// model: model name/ID for the provider
func NewProvider(providerType, apiKey, model string) (AIProvider, error) {
	return nil, fmt.Errorf("not implemented yet")
}
