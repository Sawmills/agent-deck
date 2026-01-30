package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAnthropicProvider_Chat(t *testing.T) {
	// This test uses a local HTTP server to verify request shaping and response parsing
	// for the Anthropic provider, and ensures unsupported roles fail before any network call.
	t.Run("success", func(t *testing.T) {
		var calls atomic.Int32
		reqErr := make(chan error, 1)
		recordErr := func(err error) {
			if err == nil {
				return
			}
			select {
			case reqErr <- err:
			default:
			}
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != http.MethodPost {
				recordErr(fmt.Errorf("unexpected method: %s", r.Method))
			}
			if r.URL.Path != "/v1/messages" {
				recordErr(fmt.Errorf("unexpected path: %s", r.URL.Path))
			}
			defer r.Body.Close()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				recordErr(fmt.Errorf("read body: %w", err))
			}

			var payload struct {
				Model     string `json:"model"`
				MaxTokens int    `json:"max_tokens"`
				Messages  []struct {
					Role    string `json:"role"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				recordErr(fmt.Errorf("decode body: %w", err))
			}
			if payload.Model != "claude-test" {
				recordErr(fmt.Errorf("unexpected model: %s", payload.Model))
			}
			if payload.MaxTokens == 0 || len(payload.Messages) != 1 {
				recordErr(fmt.Errorf("unexpected message payload"))
			}
			if payload.Messages[0].Role != "user" || payload.Messages[0].Content[0].Text != "hello" {
				recordErr(fmt.Errorf("unexpected message content"))
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
  "id": "msg_1",
  "type": "message",
  "role": "assistant",
  "model": "claude-test",
  "stop_reason": "end_turn",
  "stop_sequence": "",
  "content": [{"type": "text", "text": "hello"}],
  "usage": {
    "cache_creation": {
      "ephemeral_1h_input_tokens": 0,
      "ephemeral_5m_input_tokens": 0
    },
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "input_tokens": 1,
    "output_tokens": 1,
    "server_tool_use": {"web_search_requests": 0},
    "service_tier": "standard"
  }
}`))
		}))
		defer ts.Close()

		t.Setenv("ANTHROPIC_BASE_URL", ts.URL)
		provider, err := NewAnthropicProvider("test-key", "claude-test")
		if err != nil {
			t.Fatalf("NewAnthropicProvider() error: %v", err)
		}

		response, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
		if err != nil {
			t.Fatalf("Chat() error: %v", err)
		}
		if response != "hello" {
			t.Fatalf("unexpected response: %s", response)
		}
		if calls.Load() != 1 {
			t.Fatalf("expected 1 request, got %d", calls.Load())
		}
		select {
		case err := <-reqErr:
			t.Fatalf("request validation failed: %v", err)
		default:
		}
	})

	t.Run("unsupported-role", func(t *testing.T) {
		// This subtest ensures validation happens before any external call is attempted.
		provider, err := NewAnthropicProvider("test-key", "claude-test")
		if err != nil {
			t.Fatalf("NewAnthropicProvider() error: %v", err)
		}

		_, err = provider.Chat(context.Background(), []Message{{Role: "system", Content: "noop"}})
		if err == nil {
			t.Fatal("expected error for unsupported message role")
		}
		if !strings.Contains(err.Error(), "unsupported message role") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestOpenAIProvider_Chat(t *testing.T) {
	// This test verifies the OpenAI provider maps messages correctly and extracts
	// the assistant response from a mocked chat completion endpoint.
	var calls atomic.Int32
	reqErr := make(chan error, 1)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		select {
		case reqErr <- err:
		default:
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			recordErr(fmt.Errorf("unexpected method: %s", r.Method))
		}
		if r.URL.Path != "/v1/chat/completions" {
			recordErr(fmt.Errorf("unexpected path: %s", r.URL.Path))
		}
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			recordErr(fmt.Errorf("read body: %w", err))
		}

		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			recordErr(fmt.Errorf("decode body: %w", err))
		}
		if payload.Model != "gpt-test" {
			recordErr(fmt.Errorf("unexpected model: %s", payload.Model))
		}
		if len(payload.Messages) != 1 || payload.Messages[0].Role != "user" || payload.Messages[0].Content != "hi" {
			recordErr(fmt.Errorf("unexpected message payload"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id": "chatcmpl_1",
  "object": "chat.completion",
  "created": 1,
  "model": "gpt-test",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "hello"},
      "finish_reason": "stop"
    }
  ],
  "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
}`))
	}))
	defer ts.Close()

	provider, err := NewOpenAIProvider("test-key", "gpt-test", ts.URL+"/v1")
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error: %v", err)
	}

	response, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if response != "hello" {
		t.Fatalf("unexpected response: %s", response)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", calls.Load())
	}
	select {
	case err := <-reqErr:
		t.Fatalf("request validation failed: %v", err)
	default:
	}
}

func TestRetry(t *testing.T) {
	// This test forces three failures and a final success to confirm retry count
	// and backoff timing between attempts.
	var callTimes []time.Time
	startErr := errors.New("try again")

	err := WithRetry(func() error {
		callTimes = append(callTimes, time.Now())
		if len(callTimes) < 4 {
			return startErr
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithRetry() error: %v", err)
	}
	if len(callTimes) != 4 {
		t.Fatalf("expected 4 attempts, got %d", len(callTimes))
	}

	minDelays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	maxSlack := 1 * time.Second
	for i := 1; i < len(callTimes); i++ {
		delay := callTimes[i].Sub(callTimes[i-1])
		if delay < minDelays[i-1] {
			t.Fatalf("attempt %d delay too short: %v", i, delay)
		}
		if delay > minDelays[i-1]+maxSlack {
			t.Fatalf("attempt %d delay too long: %v", i, delay)
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	// This test triggers panics through nil receivers and verifies they are
	// converted into provider-specific errors instead of crashing the caller.
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "trigger"}}

	tests := []struct {
		name           string
		call           func() (string, error)
		expectContains string
	}{
		{
			name: "anthropic",
			call: func() (string, error) {
				var provider *AnthropicProvider
				return provider.Chat(ctx, messages)
			},
			expectContains: "panic in Anthropic provider",
		},
		{
			name: "openai",
			call: func() (string, error) {
				var provider *OpenAIProvider
				return provider.Chat(ctx, messages)
			},
			expectContains: "panic in OpenAI provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := tt.call()
			if err == nil {
				t.Fatalf("expected error from panic recovery")
			}
			if response != "" {
				t.Fatalf("expected empty response, got %q", response)
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
