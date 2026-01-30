# PROJECT KNOWLEDGE BASE

**Generated:** 2026-01-30
**Commit:** e1dd049
**Branch:** feat/opencode-session-forking

## OVERVIEW

Go 1.24 CLI + TUI for managing multiple AI agent sessions (Claude, Gemini, OpenCode, Codex). Built on Bubble Tea/Lipgloss with tmux backend. MCP socket pooling reduces memory 85-90%.

## STRUCTURE

```
agent-deck/
├── cmd/agent-deck/      # CLI entry (main.go 1639+ LOC, all commands in one file)
├── internal/
│   ├── session/         # Core engine (48 files) - session lifecycle, tool detection, MCP, storage
│   ├── ui/              # TUI layer (36 files) - Bubble Tea components, dialogs, panels
│   ├── tmux/            # tmux integration - session cache, status detection, PTY
│   ├── mcppool/         # MCP socket pooling - Unix socket proxy, HTTP server
│   ├── git/             # Worktree support
│   ├── clipboard/       # Cross-platform clipboard
│   ├── platform/        # OS detection (WSL, Windows)
│   ├── profile/         # Shell profile detection
│   ├── experiments/     # Feature flags
│   └── update/          # Auto-update logic
├── skills/              # Claude Code skill definitions (not Go code)
├── site/                # Static website assets
└── demos/               # Video demos (260MB+, consider separate repo)
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add CLI command | `cmd/agent-deck/main.go` | Single file, add handler + routing |
| Session logic | `internal/session/instance.go` | 2932 LOC, core entity |
| TUI components | `internal/ui/` | `home.go` is main model (7141 LOC) |
| Status detection | `internal/tmux/tmux.go` | `RefreshSessionCache()`, patterns in CapturePane |
| MCP management | `internal/session/mcp_catalog.go` | Discovery, .mcp.json generation |
| Configuration | `internal/session/userconfig.go` | TOML parsing, tool definitions |
| Persistence | `internal/session/storage.go` | JSON files, profile isolation |

## CONVENTIONS

**Go Standards Applied** - No custom linter config, uses golangci-lint defaults.

**Commits**: Conventional Commits required (feat/fix/docs/refactor/perf)

**Branches**: `feature/`, `fix/`, `perf/`, `docs/`, `refactor/`

**Testing**:
- Every package has `testmain_test.go` with `AGENTDECK_PROFILE=_test` isolation
- Use `skipIfNoTmuxServer(t)` for tmux-dependent tests
- Table-driven tests preferred
- Tests include detailed comments explaining intent

**Environment**:
- `AGENTDECK_DEBUG=1` enables debug logging
- `AGENTDECK_PROFILE=_test` isolates test data

## ANTI-PATTERNS (THIS PROJECT)

**CRITICAL - Data Destruction**:
- NEVER run `tmux kill-server` or `tmux kill-session` - destroys ALL agent-deck sessions irreversibly

**Shell Syntax**:
- Capture-resume commands with `$(...)` MUST use "claude" binary, not aliases
- Commands wrapped in `bash -c` for fish compatibility

**Deprecated APIs**:
- `Save()` → use `SaveWithGroups()` (groups can be lost)
- `GetStoragePathForProfile()` → use explicit profile support

**MCP Warning**:
- MCPs written to `~/.claude.json` affect ALL Claude sessions globally

**Status Detection**:
- Known flicker issue: status shows 'running' immediately after Start()
- Use grace period checks (`lastErrorCheck`, `lastStartTime`)

## UNIQUE STYLES

**Single-file CLI**: All command handlers in `main.go` rather than separate `cmd/` packages

**Tool Abstraction**: Tool-agnostic detection from files, not hardcoded:
- Claude: `~/.claude/projects/<hash>/sessions/`
- Gemini: `~/.gemini/tmp/<hash>/chats/`
- OpenCode: `~/.opencode/sessions/`

**Transient vs Persistent State**:
```go
ClaudeSessionID string `json:"claude_session_id"` // Persisted
tmuxSession *tmux.Session `json:"-"`              // NOT persisted
lastErrorCheck time.Time                           // NOT persisted
```

**Caching Strategy**:
- tmux session cache: O(1) via single `tmux list-windows` call
- Preview cache: 150ms debounce during navigation
- Analytics cache: 5s TTL

## COMMANDS

```bash
# Development
make build        # Build to ./build/agent-deck
make run          # Direct run
make dev          # Auto-reload with air
make fmt          # go fmt ./...
make lint         # golangci-lint
make test         # go test -v ./...

# Installation
make install      # System-wide (/usr/local/bin, requires sudo)
make install-user # User-local (~/.local/bin)

# Release
make release      # Cross-platform (darwin/linux × amd64/arm64)
```

## NOTES

**Profile Isolation**: Each profile (`~/.agent-deck/profiles/{name}/`) is completely independent.

**MCP Pooling**: Enable `pool_all = true` in config.toml. Disabled on WSL1/Windows (no Unix sockets).

**Backup Rotation**: Storage keeps 3 rolling backups of sessions.json.

**tmux Session Prefix**: All sessions prefixed `agentdeck_*` to avoid conflicts.

**Version Injection**: Set via `-ldflags "-X main.Version=$(VERSION)"` in Makefile.
