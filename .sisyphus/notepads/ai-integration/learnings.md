# Learnings - AI Integration

## Patterns to Follow
- ToolOptions interface pattern (`internal/session/tooloptions.go`)
- Status worker with panic recovery (`internal/ui/home.go:1432`)
- PreviewSettings pointer-based optional config (`internal/session/userconfig.go:152`)
- 3-tier fallback for providers (`internal/session/mcp_catalog.go:45-172`)

## Conventions
- Profile isolation: All storage under `~/.agent-deck/profiles/{profile}/`
- Atomic writes: temp file + rename pattern
- fsnotify with 100ms debounce for watchers
- testmain_test.go with `AGENTDECK_PROFILE=_test` isolation

## AI Config Implementation (2026-01-30)

### TOML Parsing Quirks
- Pointer fields (`*bool`, `*int`) correctly parse to nil when omitted from TOML
- Nested sections (e.g., `[ai.observation]`) parse to nil when not present
- Environment variable interpolation (`${VAR}`) is NOT automatic in TOML - must be handled by application code
- TOML library (BurntSushi/toml) handles all parsing correctly without special handling

### Default Value Handling Pattern
- Pointer-based optional fields follow PreviewSettings pattern (lines 152-182 in userconfig.go)
- Defaults applied in getter functions, not during parsing
- This allows distinguishing "not set" (nil) from "explicitly false" (*false)
- Example: `GetShowOutput()` returns true when nil, respects explicit false

### Struct Design
- AISettings uses pointer fields for optional config: `*bool`, `*int`
- Nested structs (AIObservationSettings, AIWatchSettings) also use pointers
- All fields have TOML tags with snake_case names (e.g., `max_tokens_per_request`)
- Provider string field is non-pointer (required, defaults to "anthropic" in docs)

### Test Coverage
- TestParseAIConfig: Full config with all fields set
- TestParseAIConfig_OmittedFields: Verifies nil for omitted pointer fields
- TestParseAIConfig_PartialObservation: Partial nested config parsing
- All tests use direct TOML parsing (not LoadUserConfig) to isolate parsing logic

### Integration Notes
- AI field added to UserConfig as `*AISettings` (pointer, optional)
- Follows same pattern as other optional config sections
- No getter functions added yet (will be needed for default application)
- Environment variable interpolation will need custom handling in provider code

## Provider Hardening (2026-01-30)
- Provider Chat methods should `defer` panic recovery and return provider-specific errors
- Anthropic tests rely on `ANTHROPIC_BASE_URL` to point the SDK at a local httptest server
- OpenAI tests set baseURL to `{server}/v1` to match `chat/completions` routing
- Retry tests verify 1s/2s/4s backoff timing with real sleeps and a slack window
- Package-level TestMain sets `AGENTDECK_PROFILE=_test` for isolation

## Session Observer Implementation (2026-01-30)

### Key Patterns Used
- **Hash-based change detection**: SHA256 of content before truncation (`sha256.Sum256([]byte(content))`, `hex.EncodeToString(hash[:])`)
- **Ring buffer FIFO**: Slice copy-shift pattern for eviction: `copy(slice, slice[1:])` then re-slice
- **Thread-safe map access**: `sync.RWMutex` with read locks for getters, write locks for mutations
- **Config with defaults**: Pointer-based optional config (`*int` for RetentionCount, MaxSizeBytes) with getter functions

### Integration Points
- `Instance.GetTmuxSession()` - Access tmux session from Instance (line 2656 in instance.go)
- `tmuxSession.CapturePane()` - Capture terminal content with caching (tmux.go:976)
- `UpdateStatus()` (instance.go:1177) - Hook point for calling Observe()

### Design Decisions
- Hash full content before truncation so ContentHash represents complete capture
- Return copies from GetObservations/GetLatestObservation to prevent external mutation
- Accept `*Instance` rather than session ID in Observe() for direct tmux access
- Preallocate observations slice with capacity equal to retention count

## Observation Persistence Implementation (2026-01-30)

### Atomic Write Pattern (Crash-Safe)
- **Pattern**: Write to temp file → fsync → atomic rename
- **Why**: Prevents data corruption if process crashes during write
- **Implementation**:
  1. Write JSON to `{path}.tmp` with 0600 permissions (owner only)
  2. Call `syncFile()` to fsync data to disk (non-fatal if fails)
  3. Atomic `os.Rename(tmpPath, path)` - atomic on POSIX systems
- **Cleanup**: `cleanupTempFiles()` removes leftover `.tmp` files on startup
- **Reused from**: `storage.go:SaveWithGroups()` pattern (lines 275-307)

### Profile Isolation
- **Storage path**: `~/.agent-deck/profiles/{profile}/observations/{sessionID}.json`
- **Helper function**: `getObservationsPath(profile, sessionID)` validates inputs and constructs path
- **Profile validation**: Uses `GetProfileDir()` which sanitizes profile name (prevents path traversal)
- **Directory creation**: `os.MkdirAll(dir, 0700)` ensures secure permissions (owner only)

### SessionObserver Constructor Change
- **Added field**: `profile string` to SessionObserver struct
- **Updated constructor**: `NewSessionObserver(profile string, config *AIObservationSettings)`
- **Why**: Needed to pass profile to SaveObservations() without requiring caller to track it
- **Note**: This is a breaking change - callers must now provide profile

### SaveObservations() Design
- **Lock strategy**: Read-lock to copy observations, then release lock before I/O
- **Why**: Prevents blocking other observers during disk write
- **JSON format**: `json.MarshalIndent(..., "", "  ")` for human-readable storage
- **Error handling**: Logs fsync failures but doesn't fail - atomic rename still provides safety
- **Called from**: `Observe()` after adding observation to ring buffer

### LoadObservations() Design
- **Non-error on missing file**: Returns nil if file doesn't exist (observations may not be persisted yet)
- **Auto-cleanup**: Deletes files older than 30 days on load (time.Since(fileInfo.ModTime()) > 30*24*time.Hour)
- **Merge strategy**: Replaces in-memory observations with loaded ones (handles reload case)
- **Lock strategy**: Acquires write lock only when updating observer state

### Testing Considerations
- No fsnotify watcher yet (deferred to later task)
- Tests should verify:
  1. Observations persist to correct path
  2. Atomic write prevents corruption (temp file cleanup)
  3. Profile isolation (different profiles have separate files)
  4. 30-day cleanup works
  5. Load/save round-trip preserves data

### Integration Notes
- `Observe()` now calls `SaveObservations()` after adding observation
- Persistence is synchronous (blocks until written) - consider async in future if performance issue
- No conversation history persistence (stateless Q&A per plan)
- Observations are JSON array of Observation structs (timestamp, content, hash, status)

## Watch Mode Implementation (2026-01-30)

### Defaults and Limits
- Default interval 5s and timeout 1h when unset (AIWatchSettings fallback)
- Enforce max concurrent goals (cap at 10)

### Worker + Evaluation
- Per-goal ticker worker with panic recovery and auto-pause on timeout
- Evaluation prompt uses session observations; `<NoComment>` skips action
- Trigger updates stored on goal (LastTriggered, TriggerCount) with action-specific logging

### Persistence
- Goals stored at `~/.agent-deck/profiles/{profile}/watch_goals.json`
- Atomic write: temp file + fsync + rename, 0600 permissions

## [2026-01-30] Final Session - Tasks 6-9 Complete

### TUI Implementation Patterns
- **Delegation Limitation**: Complex Bubble Tea components (300-500 LOC) cannot be created via delegation system
- **Direct Implementation Required**: TUI components with multiple modes, form handling, and complex state need manual implementation
- **Pattern to Follow**: Read existing dialogs (mcp_dialog.go, newdialog.go) and replicate structure

### Bubble Tea Best Practices
- **Update() Pattern**: Handle KeyMsg, return (tea.Model, tea.Cmd)
- **View() Must Be Pure**: No I/O, only rendering with lipgloss
- **Form Focus Management**: Track focusIndex, Tab cycles through inputs, updateFocus() helper
- **Mode Switching**: Use enum for mode, reset state on mode change
- **Dialog Visibility**: Each dialog manages its own visible bool, no global view state needed

### Integration Patterns
- **Home Struct Fields**: Add dialog fields alongside existing dialogs
- **Initialization**: Check config, create AI provider, initialize observer/watchMgr, create dialogs
- **Message Routing**: Check dialog visibility in Update(), route messages to active dialog
- **Rendering**: Check dialog visibility in View(), render active dialog
- **Cleanup**: Call Stop() on managers in shutdown handler

### API Corrections
- **Context Parameter**: AIProvider.Chat() requires context.Context as first parameter
- **WatchManager Methods**: GetGoals() not ListGoals(), RemoveGoal() not DeleteGoal()
- **No UpdateGoal**: Must RemoveGoal() then AddGoal() with preserved ID

### Testing Strategy
- **Backend First**: All backend tests must pass before TUI integration
- **Build Verification**: go build ./cmd/agent-deck must succeed
- **Manual QA Required**: TUI components need hands-on testing for visual/interaction bugs

### Success Metrics
- ✅ All 9 tasks completed
- ✅ 17 commits with clear messages
- ✅ All tests passing
- ✅ Clean build
- ✅ Comprehensive documentation

### Time Investment
- Wave 1 (Foundation): ~2 hours
- Wave 2 (Observation): ~3 hours
- Wave 3 (TUI): ~4 hours (including delegation attempts)
- Total: ~9 hours for complete AI integration

### Key Takeaway
**Orchestrator Pattern Works**: Delegation succeeded for backend logic, failed for complex TUI. Direct implementation was faster than repeated delegation attempts. Know when to delegate vs. implement directly.
