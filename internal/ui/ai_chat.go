package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/ai"
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AIChatPanel is a Bubble Tea component for AI chat about session content.
// TODO: Implement full chat functionality with streaming responses
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
}

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// NewAIChatPanel creates a new AI chat panel
func NewAIChatPanel(sessionID string, observer *session.SessionObserver, aiProvider ai.AIProvider) *AIChatPanel {
	ti := textinput.New()
	ti.Placeholder = "Ask about this session..."
	ti.Focus()

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
	// TODO: Implement full update logic with:
	// - Enter key to send message
	// - Esc key to close
	// - Ctrl+L to clear history
	// - Streaming response handling

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			p.visible = false
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

// View implements tea.Model
func (p AIChatPanel) View() string {
	if !p.visible {
		return ""
	}

	// TODO: Implement full view with:
	// - Message history display
	// - Lipgloss styling
	// - Loading indicator
	// - Context from observations

	return "AI Chat Panel (TODO: Implement)\n\n" + p.input.View()
}

// SetSize updates the panel dimensions
func (p *AIChatPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}
