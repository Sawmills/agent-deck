# Draft: AI Integration for Agent Deck (TmuxAI Capabilities)

## Requirements (confirmed)

- **Track sessions**: Observe terminal content, detect what AI agents are doing
- **Ask questions**: Chat with AI about session content ("what's happening?")
- **Manage them**: AI-assisted session control (suggestions, proactive alerts)

## Technical Decisions

- **Primary AI**: Claude Opus 4.5 (Anthropic)
- **Multi-provider**: Yes — Anthropic, OpenAI, OpenRouter
- **Observation storage**: Persistent (JSON files in ~/.agent-deck/observations/)
- **Watch scope**: Flexible — can watch all sessions OR specific sessions
- **Multiple watch goals**: Yes — concurrent watch tasks with different scopes
- **Cost control**: Token limits per request and daily

## Research Findings

### Agent Deck Existing Infrastructure (from explore agents)
- `CapturePane()` — already captures pane content with 500ms cache
- `CaptureFullHistory()` — gets last 2000 lines
- `HasUpdated()` — SHA256 hash-based change detection
- `SendKeys()`, `SendKeysChunked()` — send commands to sessions
- `LogWatcher` — fsnotify-based event-driven updates
- `UpdateStatus()` — called every 500ms, hooks into tool-specific updates
- Extension pattern: tool-specific fields on Instance struct (ClaudeSessionID, GeminiSessionID, etc.)

### TmuxAI Patterns (from librarian agents)
- Polling-based watch mode with configurable interval (default 5s)
- "NoComment" pattern — AI decides when to speak up vs stay silent
- Natural language goals evaluated by LLM reasoning
- Visible countdown for user control (pause/resume)
- XML context format for pane content

### Go AI Client Libraries (from librarian agents)
- `sashabaranov/go-openai` (10.5k stars) — OpenAI/OpenRouter
- `anthropics/anthropic-sdk-go` (official) — Anthropic Claude
- Interface pattern for multi-provider abstraction
- Streaming via `stream.Recv()` loop with `io.EOF` termination

## Scope Boundaries

### INCLUDE
- AI provider abstraction layer (Anthropic, OpenAI, OpenRouter)
- Observation layer with persistent storage
- AI chat for asking about sessions
- Watch mode with multiple concurrent goals
- TUI integration (keybindings, chat panel, watch manager)
- Configuration in config.toml
- Token counting and daily limits

### EXCLUDE
- Auto-fix/auto-approve actions (notify/suggest only for v1)
- Voice interface
- IDE integration
- Custom fine-tuned models

## Open Questions
- None — all requirements clarified

## User Preferences
- Wants flexibility in watch scope
- Multiple concurrent watch tasks
- Persistent observations
- Token limits for cost control
