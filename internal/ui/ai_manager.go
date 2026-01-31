package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/ai"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// AIManager encapsulates AI provider state and summary generation logic.
// The aiProvider is used to generate session summaries via LLM calls.
// The aiChatPanel provides interactive AI chat about sessions.
//
// Embedded in Home for field access compatibility (same pattern as AnalyticsManager).
type AIManager struct {
	aiProvider  ai.AIProvider // AI provider for generating summaries
	aiChatPanel *AIChatPanel  // For AI chat about sessions
}

// generateAISummary returns a command that asynchronously generates an AI summary
// for the given session. Uses the configured AI provider to create a concise
// one-sentence summary of what was accomplished in the session.
//
// Returns nil if:
// - inst is nil or aiProvider is not configured
// - summary already exists and was generated within the last 5 minutes
// - no context (todo, terminal output, or prompt) is available
func (h *Home) generateAISummary(inst *session.Instance) tea.Cmd {
	if inst == nil || h.aiProvider == nil {
		return nil
	}

	if inst.AISummary != "" && time.Since(inst.AISummaryGeneratedAt) < 5*time.Minute {
		return nil
	}

	sessionID := inst.ID
	tool := inst.Tool
	title := inst.Title

	todoContext := inst.GetOpenCodeTodoContext()
	terminalOutput, _ := inst.Preview()

	if todoContext == "" && terminalOutput == "" && inst.LatestPrompt == "" {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		systemPrompt := fmt.Sprintf(`Summarize this %s coding session in ONE sentence (max 150 chars).
Focus on WHAT was built/fixed/implemented. Be specific with endpoints, features, or components.
Do not start with "The user" or "This session" - just state the work done.
If the context is unclear, just return the session title as-is without explanation.

Good examples:
- "Added GET /v1/organizations/{org_id} endpoint with membership validation and tests"
- "Fixed Docker build failures in CI pipeline by updating base image"
- "Implemented dark mode toggle with localStorage persistence"`, tool)

		var contextParts []string
		contextParts = append(contextParts, fmt.Sprintf("Session title: %s", title))

		if todoContext != "" {
			contextParts = append(contextParts, todoContext)
		} else if terminalOutput != "" {
			lines := strings.Split(terminalOutput, "\n")
			if len(lines) > 50 {
				lines = lines[len(lines)-50:]
			}
			contextParts = append(contextParts, fmt.Sprintf("Recent terminal output:\n%s", strings.Join(lines, "\n")))
		}

		userPrompt := strings.Join(contextParts, "\n\n")
		combinedPrompt := systemPrompt + "\n\n" + userPrompt

		response, err := h.aiProvider.Chat(ctx, []ai.Message{
			{Role: "user", Content: combinedPrompt},
		})
		if err != nil {
			return aiSummaryMsg{sessionID: sessionID, err: err}
		}

		summary := strings.TrimSpace(response)
		if len(summary) > 160 {
			summary = summary[:157] + "..."
		}

		return aiSummaryMsg{sessionID: sessionID, summary: summary}
	}
}
