# AI Integration for Agent Deck (TmuxAI Capabilities)

## TL;DR

> **Quick Summary**: Add AI-powered session tracking, chat, and watch mode to Agent Deck — observe what sessions are doing, ask questions about them, and set up proactive monitoring goals.
> 
> **Deliverables**:
> - AI provider abstraction (Anthropic primary, OpenAI/OpenRouter support)
> - Persistent observation layer with FIFO retention
> - AI chat panel for asking questions about sessions
> - Watch mode with multiple concurrent goals
> - TUI integration with keybindings and dialogs
> 
> **Estimated Effort**: Large (7-10 days)
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Task 1 → Task 3 → Task 5 → Task 7 → Task 9

---

## Context

### Original Request
Integrate TmuxAI-style capabilities into Agent Deck:
- Track sessions (observe terminal content)
- Ask questions (AI chat about session content)
- Manage them (proactive watch goals with multiple concurrent tasks)

### Interview Summary
**Key Discussions**:
- Primary AI: Claude Opus 4.5, multi-provider support (Anthropic, OpenAI, OpenRouter)
- Observation storage: Persistent JSON with FIFO retention
- Watch scope: Flexible — all sessions or specific sessions per goal
- Multiple concurrent watch goals with different scopes/intervals
- Cost control via token limits

**Research Findings**:
- Agent Deck has `CapturePane()`, `HasUpdated()`, `SendKeys()`, `LogWatcher` already
- TmuxAI uses polling with NoComment pattern for watch mode
- Go libraries: `sashabaranov/go-openai`, `anthropics/anthropic-sdk-go`
- Extension pattern: follow `ToolOptions` interface for provider abstraction

### Metis Review
**Identified Gaps** (addressed):
- Retry strategy: Exponential backoff with 3 retries
- Observation retention: 100 per session, FIFO eviction, 50KB max
- Watch goal timeout: 1 hour default, configurable
- Goal persistence: Survives TUI restart
- TUI responsiveness: Async pattern with buffered channels
- Hard limits: Enforced from day 1

---

## Work Objectives

### Core Objective
Enable users to observe, question, and proactively monitor their AI coding sessions through an integrated AI layer.

### Concrete Deliverables
- `internal/ai/` package with provider abstraction
- `internal/session/observer.go` for observation layer
- `internal/session/watch.go` for watch mode
- `internal/ui/ai_chat.go` for chat panel
- `internal/ui/watch_dialog.go` for watch goal management
- Configuration in `config.toml` under `[ai]` section
- Persistent storage in `~/.agent-deck/profiles/{profile}/observations/`

### Definition of Done
- [ ] `agent-deck` TUI shows AI chat panel when pressing `A`
- [ ] Watch goals can be created, listed, paused, and deleted
- [ ] Observations persist across TUI restart
- [ ] Token limits are enforced and surfaced in config
- [ ] All tests pass: `make test`
- [ ] No lint errors: `make lint`

### Must Have
- AI provider abstraction with Anthropic as primary
- Streaming responses for chat
- Persistent observations with FIFO retention
- Multiple concurrent watch goals
- Token limits per request and daily
- Panic recovery in all AI goroutines

### Must NOT Have (Guardrails)
- No conversation persistence (v1 is stateless Q&A)
- No automated actions from watch goals (notify/suggest only)
- No regex in watch scope (simple list matching only)
- No provider cost tracking or comparison
- No observation search or export
- No function calling / tool use (just chat)

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES (Go test with `make test`)
- **User wants tests**: YES (TDD for core logic)
- **Framework**: Go testing + testify assertions

### Test Commands
```bash
# Unit tests
go test -v ./internal/ai/...
go test -v ./internal/session/observer_test.go
go test -v ./internal/session/watch_test.go

# Integration test
make test

# Lint
make lint
```

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately):
├── Task 1: AI Provider abstraction (internal/ai/)
└── Task 2: Configuration schema ([ai] section)

Wave 2 (After Wave 1):
├── Task 3: Observation layer (internal/session/observer.go)
├── Task 4: Observation storage (persistence)
└── Task 5: Watch mode core (internal/session/watch.go)

Wave 3 (After Wave 2):
├── Task 6: AI Chat TUI (internal/ui/ai_chat.go)
├── Task 7: Watch dialog TUI (internal/ui/watch_dialog.go)
└── Task 8: Keybinding integration (home.go)

Final (After Wave 3):
└── Task 9: Integration testing and polish
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 1 | None | 3, 5, 6 | 2 |
| 2 | None | 3, 5 | 1 |
| 3 | 1, 2 | 6 | 4, 5 |
| 4 | 1, 2 | 6 | 3, 5 |
| 5 | 1, 2 | 7 | 3, 4 |
| 6 | 3, 4 | 8 | 7 |
| 7 | 5 | 8 | 6 |
| 8 | 6, 7 | 9 | None |
| 9 | 8 | None | None |

---

## TODOs

### Task 1: AI Provider Abstraction ✅

**What to do**:
- Create `internal/ai/` package
- Define `AIProvider` interface with `Chat()` and `ChatStream()` methods
- Implement `AnthropicProvider` using `anthropics/anthropic-sdk-go`
- Implement `OpenAIProvider` using `sashabaranov/go-openai` (works for OpenRouter too)
- Add factory function `NewProvider(config) AIProvider`
- Implement exponential backoff retry (3 retries, 1s/2s/4s delays)
- Add panic recovery wrapper for all API calls

**Must NOT do**:
- No function calling / tool use
- No conversation history management
- No cost tracking

**Recommended Agent Profile**:
- **Category**: `ultrabrain`
  - Reason: Core abstraction layer requires careful interface design
- **Skills**: [`systematic-debugging`]
  - `systematic-debugging`: API integration can have subtle issues

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 1 (with Task 2)
- **Blocks**: Tasks 3, 5, 6
- **Blocked By**: None

**References**:
- `internal/session/tooloptions.go` — Follow interface pattern for provider abstraction
- `internal/session/mcp_catalog.go:45-172` — Follow 3-tier fallback pattern
- `github.com/anthropics/anthropic-sdk-go` — Official Anthropic SDK
- `github.com/sashabaranov/go-openai` — OpenAI/OpenRouter SDK

**Acceptance Criteria**:

```bash
# Unit test: Provider returns response
go test -v ./internal/ai/... -run TestAnthropicProvider
# Assert: PASS

# Unit test: Retry on failure
go test -v ./internal/ai/... -run TestRetryOnFailure
# Assert: PASS, shows 3 retry attempts in log

# Unit test: Panic recovery
go test -v ./internal/ai/... -run TestPanicRecovery
# Assert: PASS, no panic propagates
```

**Commit**: YES
- Message: `feat(ai): add AI provider abstraction layer`
- Files: `internal/ai/*.go`
- Pre-commit: `go test ./internal/ai/...`

---

### Task 2: Configuration Schema ✅

**What to do**:
- Add `[ai]` section to `UserConfig` struct in `userconfig.go`
- Add `AISettings` struct with fields:
  - `Enabled *bool` (default: false)
  - `Provider string` (anthropic, openai, openrouter)
  - `APIKey string` (env var interpolation)
  - `Model string` (default: claude-opus-4-5-20250514)
  - `MaxTokensPerRequest *int` (default: 4096)
  - `DailyTokenLimit *int` (default: 100000)
  - `RequestTimeout *int` (seconds, default: 30)
- Add `AIObservationSettings` struct:
  - `Persist *bool` (default: true)
  - `RetentionCount *int` (default: 100)
  - `MaxSizeBytes *int` (default: 51200 = 50KB)
- Add `AIWatchSettings` struct:
  - `Enabled *bool` (default: true)
  - `MaxConcurrentGoals *int` (default: 10)
  - `DefaultInterval *int` (seconds, default: 5)
  - `DefaultTimeout *int` (seconds, default: 3600 = 1 hour)
- Add `WatchGoal` struct for persisted goals
- Use pointer types for optional fields (match `PreviewSettings` pattern)

**Must NOT do**:
- No provider-specific config sections (keep it simple)
- No cost tracking fields

**Recommended Agent Profile**:
- **Category**: `quick`
  - Reason: Straightforward struct additions following existing patterns
- **Skills**: []

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 1 (with Task 1)
- **Blocks**: Tasks 3, 5
- **Blocked By**: None

**References**:
- `internal/session/userconfig.go:152-182` — `PreviewSettings` pattern with pointer fields
- `internal/session/userconfig.go:76-114` — `MCPPoolSettings` for nested config example

**Acceptance Criteria**:

```bash
# Parse config with new [ai] section
cat > /tmp/test-config.toml << 'EOF'
[ai]
enabled = true
provider = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"
model = "claude-opus-4-5-20250514"
max_tokens_per_request = 4096

[ai.observation]
persist = true
retention_count = 100

[ai.watch]
enabled = true
max_concurrent_goals = 10
EOF

go test -v ./internal/session/... -run TestParseAIConfig
# Assert: PASS, config parsed correctly
```

**Commit**: YES
- Message: `feat(config): add AI settings schema`
- Files: `internal/session/userconfig.go`
- Pre-commit: `go test ./internal/session/...`

---

### Task 3: Observation Layer ✅

**What to do**:
- Create `internal/session/observer.go`
- Define `SessionObserver` struct with:
  - `sessions map[string]*ObservedSession`
  - `aiProvider ai.AIProvider`
  - `config *AIObservationSettings`
- Define `ObservedSession` struct with:
  - `Instance *Instance`
  - `Observations []Observation` (ring buffer)
  - `LastObserved time.Time`
  - `ContentHash string`
- Define `Observation` struct with:
  - `Timestamp time.Time`
  - `Content string` (truncated to MaxSizeBytes)
  - `ContentHash string`
  - `Status Status`
- Implement `Observe(sessionID)` method:
  - Call `CapturePane()` on session
  - Compare hash to detect changes
  - Truncate to MaxSizeBytes if needed
  - Add to ring buffer (FIFO eviction at RetentionCount)
- Hook into `UpdateStatus()` to call `Observe()` for tracked sessions

**Must NOT do**:
- No search or query functionality
- No export functionality
- No observation analytics

**Recommended Agent Profile**:
- **Category**: `unspecified-high`
  - Reason: Core data layer with state management
- **Skills**: [`systematic-debugging`]
  - `systematic-debugging`: State management can have subtle bugs

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 2 (with Tasks 4, 5)
- **Blocks**: Task 6
- **Blocked By**: Tasks 1, 2

**References**:
- `internal/session/instance.go:1177` — `UpdateStatus()` hook point
- `internal/tmux/tmux.go:976` — `CapturePane()` implementation
- `internal/tmux/tmux.go:1033` — `HasUpdated()` hash comparison pattern

**Acceptance Criteria**:

```bash
# Unit test: Observation recorded
go test -v ./internal/session/... -run TestObservationRecorded
# Assert: PASS

# Unit test: FIFO eviction at limit
go test -v ./internal/session/... -run TestObservationFIFOEviction
# Assert: PASS, max 100 observations

# Unit test: Content truncated at max size
go test -v ./internal/session/... -run TestObservationTruncation
# Assert: PASS, content <= 50KB
```

**Commit**: YES
- Message: `feat(session): add observation layer for session tracking`
- Files: `internal/session/observer.go`, `internal/session/observer_test.go`
- Pre-commit: `go test ./internal/session/...`

---

### Task 4: Observation Persistence ✅

**What to do**:
- Create storage path: `~/.agent-deck/profiles/{profile}/observations/{sessionID}.json`
- Implement `SaveObservations(sessionID)` — atomic write (temp + rename)
- Implement `LoadObservations(sessionID)` — load on demand
- Add fsnotify watcher with 100ms debounce (match `storage_watcher.go` pattern)
- Implement retention cleanup on load (delete files older than 30 days)
- Profile-isolate all storage (match sessions.json pattern)

**Must NOT do**:
- No cross-profile observation access
- No observation export

**Recommended Agent Profile**:
- **Category**: `quick`
  - Reason: Follow existing storage patterns closely
- **Skills**: []

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 2 (with Tasks 3, 5)
- **Blocks**: Task 6
- **Blocked By**: Tasks 1, 2

**References**:
- `internal/session/storage.go` — Profile isolation, atomic writes, backup rotation
- `internal/tmux/watcher.go` — fsnotify with rate limiting pattern

**Acceptance Criteria**:

```bash
# Unit test: Observations persist to disk
go test -v ./internal/session/... -run TestObservationPersistence
# Assert: PASS, file exists at expected path

# Unit test: Atomic write (no corruption on crash)
go test -v ./internal/session/... -run TestObservationAtomicWrite
# Assert: PASS, uses temp file + rename

# Unit test: Profile isolation
go test -v ./internal/session/... -run TestObservationProfileIsolation
# Assert: PASS, observations stored under correct profile
```

**Commit**: YES
- Message: `feat(session): add persistent observation storage`
- Files: `internal/session/observer.go` (extended)
- Pre-commit: `go test ./internal/session/...`

---

### Task 5: Watch Mode Core

**What to do**:
- Create `internal/session/watch.go`
- Define `WatchManager` struct with:
  - `goals map[string]*WatchGoal`
  - `observer *SessionObserver`
  - `aiProvider ai.AIProvider`
  - `config *AIWatchSettings`
- Define `WatchGoal` struct with:
  - `ID string`
  - `Name string`
  - `Description string` (natural language goal)
  - `Sessions []string` (empty = all, or specific IDs/patterns)
  - `Interval time.Duration`
  - `Timeout time.Duration`
  - `Action WatchAction` (notify, log, suggest)
  - `Paused bool`
  - `CreatedAt time.Time`
  - `LastTriggered time.Time`
  - `TriggerCount int`
- Implement polling loop following `home.go:1432` status worker pattern:
  - Ticker at goal interval
  - Panic recovery
  - Context cancellation on shutdown
- Implement `evaluateGoal()` with NoComment pattern:
  - Build prompt with session content + goal description
  - If AI returns `<NoComment>`, skip
  - Otherwise, trigger action
- Implement goal persistence to `~/.agent-deck/profiles/{profile}/watch_goals.json`
- Enforce `MaxConcurrentGoals` limit

**Must NOT do**:
- No automated actions (restart, send keys)
- No regex matching for sessions
- No complex query language

**Recommended Agent Profile**:
- **Category**: `ultrabrain`
  - Reason: Complex async worker with multiple goroutines
- **Skills**: [`systematic-debugging`]
  - `systematic-debugging`: Concurrent code needs careful debugging

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 2 (with Tasks 3, 4)
- **Blocks**: Task 7
- **Blocked By**: Tasks 1, 2

**References**:
- `internal/ui/home.go:1432-1465` — Status worker pattern with panic recovery
- `internal/tmux/watcher.go` — Rate limiting pattern
- TmuxAI `internal/process_message.go:289-312` — Watch mode loop (from research)

**Acceptance Criteria**:

```bash
# Unit test: Goal triggers on match
go test -v ./internal/session/... -run TestWatchGoalTriggers
# Assert: PASS

# Unit test: NoComment pattern skips
go test -v ./internal/session/... -run TestWatchNoComment
# Assert: PASS, no trigger when AI returns NoComment

# Unit test: Max concurrent goals enforced
go test -v ./internal/session/... -run TestWatchMaxGoals
# Assert: PASS, error when exceeding 10

# Unit test: Goal timeout expires
go test -v ./internal/session/... -run TestWatchGoalTimeout
# Assert: PASS, goal auto-disabled after timeout

# Unit test: Goals persist across restart
go test -v ./internal/session/... -run TestWatchGoalPersistence
# Assert: PASS
```

**Commit**: YES
- Message: `feat(session): add watch mode with multiple concurrent goals`
- Files: `internal/session/watch.go`, `internal/session/watch_test.go`
- Pre-commit: `go test ./internal/session/...`

---

### Task 6: AI Chat TUI Panel

**What to do**:
- Create `internal/ui/ai_chat.go`
- Define `AIChatPanel` struct (Bubble Tea model):
  - `sessionID string` (which session we're asking about)
  - `input textinput.Model`
  - `messages []ChatMessage`
  - `loading bool`
  - `response strings.Builder` (for streaming)
- Implement `Init()`, `Update()`, `View()` methods
- Handle streaming responses via channel:
  - Show typing indicator during stream
  - Append chunks to response in real-time
- Build context from recent observations:
  - Last 5 observations
  - Current session status
  - Latest prompt from session
- Implement keybindings:
  - `Enter` — send message
  - `Esc` — close panel
  - `Tab` — switch to session (attach)
  - `Ctrl+L` — clear chat

**Must NOT do**:
- No conversation persistence (stateless)
- No multi-session context

**Recommended Agent Profile**:
- **Category**: `visual-engineering`
  - Reason: TUI component with user interaction
- **Skills**: [`frontend-ui-ux`]
  - `frontend-ui-ux`: Good UX for chat interface

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 3 (with Task 7)
- **Blocks**: Task 8
- **Blocked By**: Tasks 3, 4

**References**:
- `internal/ui/mcp_dialog.go` — Dialog pattern with keybindings
- `internal/ui/global_search.go` — Text input with results display
- `internal/ui/home.go:2627` — KeyMsg handling pattern

**Acceptance Criteria**:

```bash
# Unit test: Chat panel renders
go test -v ./internal/ui/... -run TestAIChatPanelRender
# Assert: PASS

# Unit test: Message sent on Enter
go test -v ./internal/ui/... -run TestAIChatSendMessage
# Assert: PASS

# Unit test: Panel closes on Esc
go test -v ./internal/ui/... -run TestAIChatClose
# Assert: PASS

# Manual verification via playwright:
# 1. Launch agent-deck
# 2. Select a session
# 3. Press 'A' to open AI chat
# 4. Type "What is this session doing?" and press Enter
# 5. Assert: Response appears within 30s
# 6. Press Esc to close
# Screenshot: .sisyphus/evidence/task-6-ai-chat.png
```

**Commit**: YES
- Message: `feat(ui): add AI chat panel for session questions`
- Files: `internal/ui/ai_chat.go`, `internal/ui/ai_chat_test.go`
- Pre-commit: `go test ./internal/ui/...`

---

### Task 7: Watch Dialog TUI

**What to do**:
- Create `internal/ui/watch_dialog.go`
- Define `WatchDialog` struct (Bubble Tea model):
  - `goals []*WatchGoal`
  - `cursor int`
  - `mode` (list, create, edit)
  - `form` (for create/edit)
- Implement list view:
  - Show all goals with status (active/paused)
  - Show trigger count and last triggered
- Implement create form:
  - Name input
  - Description input (the goal text)
  - Sessions input (comma-separated or "all")
  - Interval selector
  - Action selector (notify, suggest)
- Implement keybindings:
  - `n` — new goal
  - `Space` — toggle pause
  - `e` — edit goal
  - `d` — delete goal
  - `Esc` — close dialog

**Must NOT do**:
- No regex input for sessions
- No complex scheduling

**Recommended Agent Profile**:
- **Category**: `visual-engineering`
  - Reason: TUI dialog with forms and lists
- **Skills**: [`frontend-ui-ux`]
  - `frontend-ui-ux`: Good UX for form interactions

**Parallelization**:
- **Can Run In Parallel**: YES
- **Parallel Group**: Wave 3 (with Task 6)
- **Blocks**: Task 8
- **Blocked By**: Task 5

**References**:
- `internal/ui/mcp_dialog.go` — Toggle list pattern
- `internal/ui/newdialog.go` — Form with multiple inputs
- `internal/ui/group_dialog.go` — CRUD operations pattern

**Acceptance Criteria**:

```bash
# Unit test: Dialog renders goals
go test -v ./internal/ui/... -run TestWatchDialogRender
# Assert: PASS

# Unit test: Goal created via form
go test -v ./internal/ui/... -run TestWatchDialogCreate
# Assert: PASS

# Unit test: Goal toggled with Space
go test -v ./internal/ui/... -run TestWatchDialogToggle
# Assert: PASS

# Manual verification via playwright:
# 1. Launch agent-deck
# 2. Press 'W' to open watch dialog
# 3. Press 'n' to create new goal
# 4. Fill form: name="test-watch", description="alert on errors", sessions="all"
# 5. Press Enter to save
# 6. Assert: Goal appears in list
# Screenshot: .sisyphus/evidence/task-7-watch-dialog.png
```

**Commit**: YES
- Message: `feat(ui): add watch goal management dialog`
- Files: `internal/ui/watch_dialog.go`, `internal/ui/watch_dialog_test.go`
- Pre-commit: `go test ./internal/ui/...`

---

### Task 8: Keybinding Integration

**What to do**:
- Add keybindings to `internal/ui/home.go`:
  - `A` — open AI chat for selected session
  - `W` — open watch dialog
  - `Shift+W` — quick-add watch goal for selected session
- Add new view states:
  - `viewAIChat`
  - `viewWatchDialog`
- Wire up key handlers in `handleMainKey()`
- Initialize `SessionObserver` and `WatchManager` in Home model
- Start watch workers on TUI init
- Clean shutdown with WaitGroup on quit

**Must NOT do**:
- No keybinding conflicts with existing keys

**Recommended Agent Profile**:
- **Category**: `unspecified-high`
  - Reason: Integration touches many parts of home.go
- **Skills**: []

**Parallelization**:
- **Can Run In Parallel**: NO
- **Parallel Group**: Sequential (after Wave 3)
- **Blocks**: Task 9
- **Blocked By**: Tasks 6, 7

**References**:
- `internal/ui/home.go:2965` — `handleMainKey()` function
- `internal/ui/home.go:47` — View state constants
- `internal/ui/home.go:1432` — Worker initialization pattern

**Acceptance Criteria**:

```bash
# Unit test: A key opens AI chat
go test -v ./internal/ui/... -run TestHomeAIChat
# Assert: PASS

# Unit test: W key opens watch dialog
go test -v ./internal/ui/... -run TestHomeWatchDialog
# Assert: PASS

# Integration test: Full flow
make test
# Assert: All tests pass

# Manual verification via playwright:
# 1. Launch agent-deck
# 2. Press 'A' — AI chat opens
# 3. Press Esc — chat closes
# 4. Press 'W' — watch dialog opens
# 5. Press Esc — dialog closes
# Screenshot: .sisyphus/evidence/task-8-keybindings.png
```

**Commit**: YES
- Message: `feat(ui): integrate AI chat and watch mode keybindings`
- Files: `internal/ui/home.go`
- Pre-commit: `make test`

---

### Task 9: Integration Testing and Polish

**What to do**:
- Create integration test file `internal/ai/integration_test.go`
- Test full flow:
  1. Create session
  2. Start observation
  3. Open AI chat, ask question
  4. Create watch goal
  5. Trigger watch goal
  6. Verify notification
- Add help text for new keybindings in `internal/ui/help.go`
- Update README with AI features documentation
- Test with real API keys (manual)
- Performance test: 10 sessions, 1000 observations, measure memory

**Must NOT do**:
- No new features (polish only)

**Recommended Agent Profile**:
- **Category**: `unspecified-low`
  - Reason: Testing and documentation
- **Skills**: []

**Parallelization**:
- **Can Run In Parallel**: NO
- **Parallel Group**: Final
- **Blocks**: None (final task)
- **Blocked By**: Task 8

**References**:
- `internal/session/instance_test.go` — Integration test patterns
- `internal/ui/help.go` — Help text format
- `README.md` — Documentation format

**Acceptance Criteria**:

```bash
# Integration tests pass
go test -v ./internal/ai/... -run Integration
# Assert: PASS

# Full test suite
make test
# Assert: All tests pass

# Lint clean
make lint
# Assert: No errors

# Build succeeds
make build
# Assert: Exit code 0

# Memory test (manual):
# 1. Create 10 sessions
# 2. Enable observation for all
# 3. Generate 100 observations per session
# 4. Check memory: ps aux | grep agent-deck | awk '{print $6}'
# Assert: Memory < 500MB
```

**Commit**: YES
- Message: `test(ai): add integration tests and polish`
- Files: `internal/ai/integration_test.go`, `internal/ui/help.go`, `README.md`
- Pre-commit: `make test && make lint`

---

## Commit Strategy

| After Task | Message | Files | Verification |
|------------|---------|-------|--------------|
| 1 | `feat(ai): add AI provider abstraction layer` | `internal/ai/*.go` | `go test ./internal/ai/...` |
| 2 | `feat(config): add AI settings schema` | `internal/session/userconfig.go` | `go test ./internal/session/...` |
| 3 | `feat(session): add observation layer for session tracking` | `internal/session/observer.go` | `go test ./internal/session/...` |
| 4 | `feat(session): add persistent observation storage` | `internal/session/observer.go` | `go test ./internal/session/...` |
| 5 | `feat(session): add watch mode with multiple concurrent goals` | `internal/session/watch.go` | `go test ./internal/session/...` |
| 6 | `feat(ui): add AI chat panel for session questions` | `internal/ui/ai_chat.go` | `go test ./internal/ui/...` |
| 7 | `feat(ui): add watch goal management dialog` | `internal/ui/watch_dialog.go` | `go test ./internal/ui/...` |
| 8 | `feat(ui): integrate AI chat and watch mode keybindings` | `internal/ui/home.go` | `make test` |
| 9 | `test(ai): add integration tests and polish` | `internal/ai/integration_test.go` | `make test && make lint` |

---

## Success Criteria

### Verification Commands
```bash
# Full build
make build
# Expected: Exit code 0

# Full test suite
make test
# Expected: All tests pass

# Lint check
make lint
# Expected: No errors

# Run TUI and verify features manually
./build/agent-deck
# Expected: A key opens AI chat, W key opens watch dialog
```

### Final Checklist
- [ ] AI chat works with Claude Opus 4.5
- [ ] Observations persist across TUI restart
- [ ] Watch goals trigger correctly
- [ ] Token limits are enforced
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] No lint errors
