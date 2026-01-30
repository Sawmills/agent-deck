package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/ai"
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AIChatPanel is a Bubble Tea component for AI chat about session content.
type AIChatPanel struct {
	visible    bool
	sessionID  string
	observer   *session.SessionObserver
	aiProvider ai.AIProvider
	input      textinput.Model
	messages   []ChatMessage
	loading    bool
	width      int
	height     int
	err        error
}

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// aiResponseMsg is sent when AI response completes
type aiResponseMsg struct {
	content string
	err     error
}

// NewAIChatPanel creates a new AI chat panel
func NewAIChatPanel(sessionID string, observer *session.SessionObserver, aiProvider ai.AIProvider) *AIChatPanel {
	ti := textinput.New()
	ti.Placeholder = "Ask about this session..."
	ti.Focus()
	ti.Width = 60

	return &AIChatPanel{
		sessionID:  sessionID,
		observer:   observer,
		aiProvider: aiProvider,
		input:      ti,
		messages:   []ChatMessage{},
	}
}

// Show makes the panel visible
func (p *AIChatPanel) Show() {
	p.visible = true
	p.input.Focus()
}

// Hide hides the panel
func (p *AIChatPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether the panel is visible
func (p *AIChatPanel) IsVisible() bool {
	return p.visible
}

// Init implements tea.Model
func (p AIChatPanel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (p AIChatPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't process keys if loading
		if p.loading && msg.String() != "esc" {
			return p, nil
		}

		switch msg.String() {
		case "esc":
			p.visible = false
			return p, nil

		case "ctrl+l":
			// Clear chat history
			p.messages = []ChatMessage{}
			p.err = nil
			return p, nil

		case "enter":
			// Send message
			userMsg := strings.TrimSpace(p.input.Value())
			if userMsg == "" {
				return p, nil
			}

			// Add user message
			p.messages = append(p.messages, ChatMessage{
				Role:    "user",
				Content: userMsg,
			})

			// Clear input
			p.input.SetValue("")

			// Start loading
			p.loading = true
			p.err = nil

			// Send to AI
			return p, p.sendMessage(userMsg)
		}

	case aiResponseMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err
		} else {
			// Add assistant response
			p.messages = append(p.messages, ChatMessage{
				Role:    "assistant",
				Content: msg.content,
			})
		}
		return p, nil
	}

	// Update text input
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

// sendMessage sends a message to the AI provider
func (p *AIChatPanel) sendMessage(userMsg string) tea.Cmd {
	return func() tea.Msg {
		// Build context from observations
		sessionContext := p.buildContext()

		// Build full prompt
		prompt := fmt.Sprintf("Context about session %s:\n%s\n\nUser question: %s", p.sessionID, sessionContext, userMsg)

		// Call AI provider
		response, err := p.aiProvider.Chat(context.Background(), []ai.Message{
			{Role: "user", Content: prompt},
		})

		return aiResponseMsg{
			content: response,
			err:     err,
		}
	}
}

// buildContext builds context from recent observations
func (p *AIChatPanel) buildContext() string {
	if p.observer == nil {
		return "No observation data available."
	}

	observations := p.observer.GetObservations(p.sessionID)
	if len(observations) == 0 {
		return "No observations recorded for this session yet."
	}

	// Take last 5 observations
	start := 0
	if len(observations) > 5 {
		start = len(observations) - 5
	}

	var sb strings.Builder
	sb.WriteString("Recent terminal activity:\n\n")

	for i := start; i < len(observations); i++ {
		obs := observations[i]
		sb.WriteString(fmt.Sprintf("[%s] Status: %s\n", obs.Timestamp.Format("15:04:05"), obs.Status))

		// Truncate content if too long
		content := obs.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// View implements tea.Model
func (p AIChatPanel) View() string {
	if !p.visible {
		return ""
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)
	title := titleStyle.Render(fmt.Sprintf("AI Chat - Session: %s", p.sessionID))

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	// Message history
	var messageViews []string

	// Show last N messages that fit
	maxMessages := (p.height - 10) / 3 // Rough estimate: 3 lines per message
	if maxMessages < 1 {
		maxMessages = 1
	}

	start := 0
	if len(p.messages) > maxMessages {
		start = len(p.messages) - maxMessages
	}

	for i := start; i < len(p.messages); i++ {
		msg := p.messages[i]

		var msgStyle lipgloss.Style
		var prefix string

		if msg.Role == "user" {
			msgStyle = lipgloss.NewStyle().Foreground(ColorAccent)
			prefix = "You: "
		} else {
			msgStyle = lipgloss.NewStyle().Foreground(ColorGreen)
			prefix = "AI: "
		}

		// Wrap content
		content := msg.Content
		if len(content) > 80 {
			// Simple word wrap
			words := strings.Fields(content)
			var lines []string
			var currentLine string

			for _, word := range words {
				if len(currentLine)+len(word)+1 > 80 {
					lines = append(lines, currentLine)
					currentLine = word
				} else {
					if currentLine != "" {
						currentLine += " "
					}
					currentLine += word
				}
			}
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			content = strings.Join(lines, "\n    ")
		}

		messageViews = append(messageViews, msgStyle.Render(prefix+content))
	}

	messagesView := strings.Join(messageViews, "\n\n")
	if messagesView == "" {
		messagesView = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("No messages yet. Ask a question about this session!")
	}

	// Loading indicator
	var statusLine string
	if p.loading {
		statusLine = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Render("⏳ Thinking...")
	} else if p.err != nil {
		statusLine = lipgloss.NewStyle().
			Foreground(ColorRed).
			Render(fmt.Sprintf("Error: %v", p.err))
	}

	// Input field
	inputLabel := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Render("Message: ")
	inputView := inputLabel + p.input.View()

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)
	help := helpStyle.Render("Enter: send • Ctrl+L: clear • Esc: close")

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		messagesView,
		"",
		statusLine,
		"",
		inputView,
		"",
		help,
	)

	return borderStyle.Render(content)
}

// SetSize updates the panel dimensions
func (p *AIChatPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Adjust input width
	if width > 20 {
		p.input.Width = width - 20
	}
}
