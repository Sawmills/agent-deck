# SESSION MODULE

Session orchestration engine. Manages AI agent session lifecycle, tool detection, MCP configuration, persistence, and organization.

## OVERVIEW

48 files, ~15k LOC. Core module handling:
- Session lifecycle (create, start, restart, fork, kill)
- Multi-tool detection (Claude, Gemini, OpenCode, Codex)
- MCP discovery, configuration, and socket pooling
- Hierarchical group organization
- Profile-based persistence
- Global conversation search

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Session entity/lifecycle | `instance.go` | 2932 LOC, core model |
| Group hierarchy | `groups.go` | Tree operations, flattening |
| Persistence | `storage.go` | JSON, profiles, backups |
| User config | `userconfig.go` | TOML parsing, MCPs, tools |
| Claude integration | `claude.go` | Session detection, MCP info |
| Gemini integration | `gemini.go` | Session discovery, hashing |
| MCP catalog | `mcp_catalog.go` | Discovery, .mcp.json generation |
| MCP pool manager | `pool_manager.go` | Global pool init, sockets |
| Global search | `global_search.go` | JSONL indexing, fuzzy search |
| Analytics | `analytics.go` | Token counting, cost calculation |
| Notifications | `notifications.go` | Tmux status bar integration |

## PATTERNS

**Entity-Driven**: Three core entities - `Instance`, `GroupTree`, `Storage`

**Lazy Detection**: Tool sessions detected from files at runtime:
```go
// Claude: ~/.claude/projects/<hash>/sessions/
// Gemini: ~/.gemini/tmp/<hash>/chats/ (SHA256 of resolved path)
// OpenCode: ~/.opencode/sessions/
```

**Composition Over Inheritance**: Tool-specific data as fields, not subtypes:
```go
type Instance struct {
    ClaudeSessionID  string
    GeminiSessionID  string
    OpenCodeSessionID string
    // ... not separate ClaudeInstance, GeminiInstance types
}
```

**Transient vs Persistent**:
```go
ClaudeSessionID string `json:"claude_session_id"` // Persisted
tmuxSession *tmux.Session `json:"-"`              // Transient
lastStartTime time.Time                           // Transient (grace period)
```

**Tool Options as JSON**: Arbitrary tool-specific data without schema changes:
```go
ToolOptionsJSON json.RawMessage `json:"tool_options,omitempty"`
```

## CONVENTIONS

**Session ID**: Random hex (not UUID):
```go
func generateID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

**Group Path Normalization**: Forward slashes, lowercase, hyphens for spaces.

**Claude Dir Naming**: Claude replaces non-alphanumeric with hyphens. `ConvertToClaudeDirName()` must match.

**Gemini Hashing**: SHA256 of symlink-resolved absolute path.

**MCP Loaded Names**: Tracked at session start to detect pending/stale MCPs.

**Grace Periods**:
- `lastErrorCheck`: Skip Exists() for 5s after error
- `lastStartTime`: Skip checks for 2s after Start()

**Sub-Sessions**: Store parent path separately for `--add-dir` access:
```go
ParentSessionID   string  // Link to parent
ParentProjectPath string  // Grants file access
```

## ANTI-PATTERNS

- **NEVER use deprecated `Save()`** → use `SaveWithGroups()` or groups are lost
- **ALWAYS update session** before reading Gemini/Codex data (files change frequently)
- **ALWAYS set CLAUDE_SESSION_ID** in tmux env for post-restart detection
- **Global MCP config** (`~/.claude.json`) affects ALL sessions - be careful

## DEPENDENCIES

- `internal/tmux` - Session execution, status polling
- `internal/mcppool` - Socket pooling
- `internal/platform` - OS detection
- `github.com/BurntSushi/toml` - Config parsing
- `github.com/fsnotify/fsnotify` - File watching
- `github.com/sahilm/fuzzy` - Fuzzy search
