package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/asheshgoplani/agent-deck/internal/ai"
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type ContentFetcher func(sessionID string) (content string, metadata string, err error)

type AIChatPanel struct {
	visible        bool
	sessionID      string
	observer       *session.SessionObserver
	aiProvider     ai.AIProvider
	contentFetcher ContentFetcher
	input          textinput.Model
	messages       []ChatMessage
	renderedCache  map[int]string
	loading        bool
	streaming      bool
	streamContent  strings.Builder
	streamMu       sync.Mutex
	streamCancel   context.CancelFunc
	scrollOffset   int
	inputFocused   bool
	width          int
	height         int
	err            error
	mdRenderer     *glamour.TermRenderer
}

type ChatMessage struct {
	Role    string
	Content string
}

type aiResponseMsg struct {
	content string
	err     error
}

type aiStreamChunkMsg struct {
	chunk  string
	done   bool
	err    error
	chunks <-chan string
	ctx    context.Context
}

func NewAIChatPanel(sessionID string, observer *session.SessionObserver, aiProvider ai.AIProvider) *AIChatPanel {
	ti := textinput.New()
	ti.Placeholder = "Ask about this session..."
	ti.Focus()
	ti.Width = 60

	return &AIChatPanel{
		sessionID:     sessionID,
		observer:      observer,
		aiProvider:    aiProvider,
		input:         ti,
		messages:      []ChatMessage{},
		renderedCache: make(map[int]string),
		inputFocused:  true,
	}
}

func (p *AIChatPanel) SetContentFetcher(fetcher ContentFetcher) {
	p.contentFetcher = fetcher
}

func (p *AIChatPanel) Show() {
	p.visible = true
	p.inputFocused = true
	p.input.Focus()
}

func (p *AIChatPanel) Hide() {
	p.visible = false
	if p.streamCancel != nil {
		p.streamCancel()
		p.streamCancel = nil
	}
}

func (p *AIChatPanel) IsVisible() bool {
	return p.visible
}

func (p *AIChatPanel) Init() tea.Cmd {
	return textinput.Blink
}

func (p *AIChatPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if key == "esc" {
			if p.streaming && p.streamCancel != nil {
				p.streamCancel()
				p.streamCancel = nil
				p.streaming = false
				p.loading = false
				p.streamMu.Lock()
				if p.streamContent.Len() > 0 {
					p.messages = append(p.messages, ChatMessage{
						Role:    "assistant",
						Content: p.streamContent.String(),
					})
					p.streamContent.Reset()
				}
				p.streamMu.Unlock()
				p.scrollToBottom()
				return p, nil
			}
			p.visible = false
			return p, nil
		}

		if p.loading && !p.streaming {
			return p, nil
		}

		if !p.inputFocused {
			switch key {
			case "tab", "i":
				p.inputFocused = true
				p.input.Focus()
				return p, nil
			case "j", "down":
				p.scrollDown(1)
				return p, nil
			case "k", "up":
				p.scrollUp(1)
				return p, nil
			case "ctrl+d":
				p.scrollDown(5)
				return p, nil
			case "ctrl+u":
				p.scrollUp(5)
				return p, nil
			case "g":
				p.scrollOffset = 0
				return p, nil
			case "G":
				p.scrollToBottom()
				return p, nil
			case "ctrl+l":
				p.clearChat()
				return p, nil
			}
		} else {
			switch key {
			case "tab":
				if len(p.messages) > 0 {
					p.inputFocused = false
					p.input.Blur()
				}
				return p, nil
			case "ctrl+l":
				p.clearChat()
				return p, nil
			case "enter":
				userMsg := strings.TrimSpace(p.input.Value())
				if userMsg == "" {
					return p, nil
				}

				p.messages = append(p.messages, ChatMessage{
					Role:    "user",
					Content: userMsg,
				})
				delete(p.renderedCache, len(p.messages)-1)

				p.input.SetValue("")
				p.loading = true
				p.streaming = true
				p.err = nil
				p.scrollToBottom()

				return p, p.sendStreamingMessage(userMsg)
			}
		}

	case streamStartedMsg:
		return p, p.readNextChunk(msg.chunks, msg.ctx)

	case aiStreamChunkMsg:
		if msg.err != nil {
			p.loading = false
			p.streaming = false
			p.err = msg.err
			return p, nil
		}

		if msg.done {
			p.loading = false
			p.streaming = false
			p.streamMu.Lock()
			content := p.streamContent.String()
			p.streamContent.Reset()
			p.streamMu.Unlock()

			if content != "" {
				p.messages = append(p.messages, ChatMessage{
					Role:    "assistant",
					Content: content,
				})
			}
			p.scrollToBottom()
			return p, nil
		}

		p.streamMu.Lock()
		p.streamContent.WriteString(msg.chunk)
		p.streamMu.Unlock()

		if msg.chunks != nil {
			return p, p.readNextChunk(msg.chunks, msg.ctx)
		}
		return p, nil

	case aiResponseMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err
		} else {
			p.messages = append(p.messages, ChatMessage{
				Role:    "assistant",
				Content: msg.content,
			})
			p.scrollToBottom()
		}
		return p, nil
	}

	if p.inputFocused {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, cmd
	}

	return p, nil
}

func (p *AIChatPanel) clearChat() {
	p.messages = []ChatMessage{}
	p.renderedCache = make(map[int]string)
	p.scrollOffset = 0
	p.err = nil
	p.streamMu.Lock()
	p.streamContent.Reset()
	p.streamMu.Unlock()
}

func (p *AIChatPanel) scrollUp(n int) {
	p.scrollOffset -= n
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

func (p *AIChatPanel) scrollDown(n int) {
	maxScroll := p.maxScrollOffset()
	p.scrollOffset += n
	if p.scrollOffset > maxScroll {
		p.scrollOffset = maxScroll
	}
}

func (p *AIChatPanel) scrollToBottom() {
	p.scrollOffset = p.maxScrollOffset()
}

func (p *AIChatPanel) maxScrollOffset() int {
	totalMessages := len(p.messages)
	if p.streaming {
		totalMessages++
	}
	visibleMessages := p.visibleMessageCount()
	if totalMessages <= visibleMessages {
		return 0
	}
	return totalMessages - visibleMessages
}

func (p *AIChatPanel) visibleMessageCount() int {
	availableHeight := p.height - 12
	avgMessageHeight := 4
	visible := availableHeight / avgMessageHeight
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (p *AIChatPanel) sendStreamingMessage(userMsg string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	p.streamCancel = cancel

	sessionContext := p.buildContext()
	prompt := fmt.Sprintf("Context about session %s:\n%s\n\nUser question: %s\n\nProvide a concise, well-formatted response.", p.sessionID, sessionContext, userMsg)

	return func() tea.Msg {
		chunks, err := p.aiProvider.ChatStream(ctx, []ai.Message{
			{Role: "user", Content: prompt},
		})
		if err != nil {
			return aiStreamChunkMsg{err: err}
		}

		return streamStartedMsg{chunks: chunks, ctx: ctx}
	}
}

type streamStartedMsg struct {
	chunks <-chan string
	ctx    context.Context
}

func (p *AIChatPanel) readNextChunk(chunks <-chan string, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return aiStreamChunkMsg{done: true}
		case chunk, ok := <-chunks:
			if !ok {
				return aiStreamChunkMsg{done: true}
			}
			return aiStreamChunkMsg{chunk: chunk, chunks: chunks, ctx: ctx}
		}
	}
}

func (p *AIChatPanel) buildContext() string {
	var sb strings.Builder

	if p.contentFetcher != nil {
		content, metadata, err := p.contentFetcher(p.sessionID)
		if err == nil {
			if metadata != "" {
				sb.WriteString("Session Info:\n")
				sb.WriteString(metadata)
				sb.WriteString("\n\n")
			}
			if content != "" {
				sb.WriteString("Current Terminal Output (last 100 lines):\n")
				lines := strings.Split(content, "\n")
				if len(lines) > 100 {
					lines = lines[len(lines)-100:]
				}
				sb.WriteString(strings.Join(lines, "\n"))
				sb.WriteString("\n\n")
			}
		}
	}

	if p.observer != nil {
		observations := p.observer.GetObservations(p.sessionID)
		if len(observations) > 0 {
			start := 0
			if len(observations) > 3 {
				start = len(observations) - 3
			}
			sb.WriteString("Recent Activity Log:\n")
			for i := start; i < len(observations); i++ {
				obs := observations[i]
				sb.WriteString(fmt.Sprintf("[%s] %s\n", obs.Timestamp.Format("15:04:05"), obs.Status))
			}
		}
	}

	if sb.Len() == 0 {
		return "No session context available."
	}

	return sb.String()
}

func (p *AIChatPanel) View() string {
	if !p.visible {
		return ""
	}

	contentWidth := p.width - 12
	if contentWidth < 40 {
		contentWidth = 40
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Padding(0, 1)
	title := titleStyle.Render(fmt.Sprintf("AI Chat - Session: %s", truncateID(p.sessionID, 20)))

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	allMessages := p.messages
	if p.streaming {
		p.streamMu.Lock()
		streamingContent := p.streamContent.String()
		p.streamMu.Unlock()
		if streamingContent != "" {
			allMessages = append(allMessages, ChatMessage{
				Role:    "assistant",
				Content: streamingContent,
			})
		}
	}

	var messageViews []string
	visibleCount := p.visibleMessageCount()
	start := p.scrollOffset
	end := start + visibleCount
	if end > len(allMessages) {
		end = len(allMessages)
	}
	if start > len(allMessages) {
		start = len(allMessages)
	}

	hasMoreAbove := start > 0
	hasMoreBelow := end < len(allMessages)

	if hasMoreAbove {
		indicator := lipgloss.NewStyle().Foreground(ColorTextDim).Render(fmt.Sprintf("  ▲ %d more above", start))
		messageViews = append(messageViews, indicator)
	}

	for i := start; i < end; i++ {
		msg := allMessages[i]
		isStreaming := p.streaming && i == len(allMessages)-1

		if msg.Role == "user" {
			userStyle := lipgloss.NewStyle().Foreground(ColorAccent)
			messageViews = append(messageViews, userStyle.Render("You: "+msg.Content))
		} else {
			var rendered string
			if isStreaming {
				rendered = p.renderMarkdown(msg.Content, contentWidth)
			} else if cached, ok := p.renderedCache[i]; ok {
				rendered = cached
			} else {
				rendered = p.renderMarkdown(msg.Content, contentWidth)
				p.renderedCache[i] = rendered
			}
			aiLabel := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render("AI:")
			if isStreaming {
				aiLabel = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render("AI: ▍")
			}
			messageViews = append(messageViews, aiLabel+"\n"+rendered)
		}
	}

	if hasMoreBelow {
		indicator := lipgloss.NewStyle().Foreground(ColorTextDim).Render(fmt.Sprintf("  ▼ %d more below", len(allMessages)-end))
		messageViews = append(messageViews, indicator)
	}

	messagesView := strings.Join(messageViews, "\n\n")
	if messagesView == "" {
		messagesView = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("No messages yet. Ask a question about this session!")
	}

	var statusLine string
	if p.loading && !p.streaming {
		statusLine = lipgloss.NewStyle().Foreground(ColorYellow).Render("⏳ Connecting...")
	} else if p.streaming {
		statusLine = lipgloss.NewStyle().Foreground(ColorGreen).Render("● Streaming... (Esc to stop)")
	} else if p.err != nil {
		statusLine = lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("Error: %v", p.err))
	}

	inputLabel := lipgloss.NewStyle().Foreground(ColorTextDim).Render("Message: ")
	inputView := inputLabel + p.input.View()

	var helpText string
	if p.inputFocused {
		helpText = "Enter: send • Tab: scroll mode • Ctrl+L: clear • Esc: close"
	} else {
		helpText = "j/k: scroll • g/G: top/bottom • Ctrl+d/u: page • Tab/i: input • Esc: close"
	}
	help := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true).Render(helpText)

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

func (p *AIChatPanel) renderMarkdown(content string, width int) string {
	if p.mdRenderer == nil || p.width != width {
		p.mdRenderer, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
	}

	if p.mdRenderer == nil {
		return content
	}

	rendered, err := p.mdRenderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimSpace(rendered)
}

func (p *AIChatPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	if width > 20 {
		p.input.Width = width - 20
	}

	p.mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width-12),
	)
}

func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen-3] + "..."
}
