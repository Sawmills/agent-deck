# UI MODULE

TUI layer built on Bubble Tea + Lipgloss. Implements terminal application with real-time status, session management, and rich interactions.

## OVERVIEW

36 files. Main model `home.go` (7141 LOC) implements Bubble Tea's Model interface. 22 custom message types, 30+ keyboard commands.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Main application | `home.go` | Central model, all state |
| Styling/themes | `styles.go` | Tokyo Night palettes, init() |
| Session dialogs | `newdialog.go`, `forkdialog.go` | Form inputs, path suggestions |
| MCP management | `mcp_dialog.go` | Attach/detach per session |
| Search | `search.go`, `global_search.go` | Local/global fuzzy search |
| Settings | `settings_panel.go` | Configuration editor |
| Analytics | `analytics_panel.go` | Token usage, costs |
| Test helpers | `test_helpers.go` | `NewTestHome()`, exposed methods |

## PATTERNS

**Elm Architecture** (Bubble Tea):
```go
Init() tea.Cmd           // Startup commands
Update(msg) (Model, Cmd) // State transitions
View() string            // Pure render (no I/O)
```

**Single Home Model**: All state centralized. Child components (dialogs) follow same pattern.

**Dialog Pattern**:
```go
type *Dialog struct {
    visible bool
    // dialog-specific state
}
func (d *Dialog) Update(msg) (*Dialog, tea.Cmd)
func (d *Dialog) View() string
func (d *Dialog) Show() / Hide() / IsVisible()
```

**Async Commands**: I/O returns `tea.Cmd` that produces messages:
```go
func (h *Home) loadSessions() tea.Msg {
    instances, groups, err := h.storage.LoadWithGroups()
    return loadSessionsMsg{instances, groups, err}
}
```

**Thread Safety**:
```go
instancesMu sync.RWMutex       // Protects instances slice
isAttaching atomic.Bool        // Prevents View() during attach
statusUpdateIndex atomic.Int32 // Round-robin updates
```

## CONVENTIONS

**Message Types**: 22 custom types for data loading, user actions, async ops, external events. Named `*Msg` suffix.

**Responsive Breakpoints**:
- `<50 cols`: Single column (list only)
- `50-79 cols`: Stacked layout
- `80+ cols`: Dual column (list + preview)

**Caching**:
- Preview: 150ms debounce, async fetch
- Analytics: 5s TTL
- Status counts: Invalidated on instance change
- String builder: Reused to reduce allocations

**Background Workers**:
- Status updates: Round-robin (5-10 per tick) to reduce CPU 90%+
- Log watcher: Event-driven status detection
- Storage watcher: fsnotify for external changes
- Log maintenance: 5-minute interval cleanup

## ANTI-PATTERNS

- **View() must be pure** - no I/O, subprocess calls, or side effects
- **Never block Update()** - use async commands for slow operations
- **Avoid full status updates** - use round-robin pattern

## KEYBOARD HANDLING

Input flows through `Update()`:
1. Check if dialog is active → route to dialog
2. Check if search is active → route to search
3. Handle global shortcuts (q, ?, etc.)
4. Handle list navigation (j/k, Enter, etc.)

## PERFORMANCE

Priority order:
1. Round-robin status (batch 5-10 sessions)
2. Atomic flags for hot paths
3. Background worker decouples from UI
4. Debounced preview fetch
5. Reusable string builder
6. Cached status counts

## DEPENDENCIES

- `internal/session` - Session model, storage
- `internal/tmux` - Status detection, log watching
- `internal/git` - Worktree operations
- `internal/clipboard` - Copy operations
- `internal/update` - Version checking
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Components (textinput)
