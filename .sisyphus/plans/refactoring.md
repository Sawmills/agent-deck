# Agent-Deck Comprehensive Refactoring

## TL;DR

> **Quick Summary**: Decompose the 7.3K LOC god object `home.go` into focused modules, add testing interfaces, and audit startup performance - all while keeping existing tests green.
> 
> **Deliverables**:
> - `home.go` reduced from 7,355 LOC to <4,000 LOC
> - 6 new files extracted from home.go
> - `SessionProvider` interface for testing
> - Performance audit document
> 
> **Estimated Effort**: Large (8-12 tasks)
> **Parallel Execution**: YES - 2 waves after prototype
> **Critical Path**: Prototype -> Parallel Extractions -> Interface -> Performance Audit

---

## Context

### Original Request
Comprehensive refactoring of agent-deck codebase for maintainability, enhanced AI feature prep, and performance improvement.

### Interview Summary
**Key Discussions**:
- **Primary pain point**: `home.go` (7,355 LOC) is overwhelming to navigate/modify
- **Feature prep**: Enhanced AI features require cleaner AI subsystem
- **Performance**: Slow startup, general audit needed
- **Test strategy**: Rely on existing tests only

**Research Findings**:
- `Home` struct has 30+ fields including 12 dialog components
- 60+ methods on Home type, many could be grouped
- Dialogs already well-extracted to separate files (good pattern)
- Thread safety patterns in place (mutexes, atomics)
- Test coverage exists in `home_test.go` (771 LOC)

### Metis Review
**Identified Gaps** (addressed):
- Verification cadence unclear → Added per-extraction verification
- Scope creep risks → Added explicit exclusions and guardrails
- LOC estimates unvalidated → Added pre-flight validation task
- Missing acceptance criteria → Added concrete bash commands
- No rollback strategy → Added git tag before starting

---

## Work Objectives

### Core Objective
Decompose `internal/ui/home.go` from 7,355 LOC into focused, maintainable modules while preserving all existing behavior and test coverage.

### Concrete Deliverables
| Deliverable | Target |
|-------------|--------|
| `internal/ui/home.go` | < 4,000 LOC |
| `internal/ui/home_keys.go` | Key handler methods |
| `internal/ui/home_workers.go` | Background workers |
| `internal/ui/preview_manager.go` | Preview subsystem |
| `internal/ui/analytics_manager.go` | Analytics subsystem |
| `internal/ui/ai_manager.go` | AI features subsystem |
| `internal/ui/home_state.go` | State/cursor management |
| `internal/session/interfaces.go` | SessionProvider interface |
| `docs/performance-audit.md` | Startup profiling results |

### Definition of Done
- [x] `go build ./...` succeeds
- [ ] `make test` passes (all existing tests) - internal/ui passes, pre-existing failures elsewhere
- [x] `make lint` passes (no new warnings) - 60 issues baseline maintained
- [ ] `wc -l internal/ui/home.go` < 4000 - ACTUAL: 5,309 (reduced 28% from 7,355)

### Must Have
- All method signatures on `Home` unchanged
- All mutexes move WITH their methods
- One extraction per commit (enables git bisect)
- Per-extraction verification (build + test + lint)

### Must NOT Have (Guardrails)
- MUST NOT: Change method behavior (only move code)
- MUST NOT: Add new abstractions beyond planned managers
- MUST NOT: Fix "small things" discovered during move (log to issues.md instead)
- MUST NOT: Touch dialog_*.go files (already well-factored)
- MUST NOT: Add interfaces beyond SessionProvider
- MUST NOT: Modify test files unless tests fail after extraction

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES (`go test`, `make test`)
- **User wants tests**: Existing tests only
- **Framework**: Go testing + testify

### Per-Extraction Verification (MANDATORY)
After EACH file extraction:
```bash
go build ./...     # Must succeed
make test          # Must pass
make lint          # Must pass (no new warnings)
```

### Regression Detection
If any extraction breaks tests:
1. `git revert` that commit immediately
2. Analyze what coupling was missed
3. DO NOT continue until issue resolved

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 0 (Pre-Flight):
└── Task 0: Validate estimates and create rollback point

Wave 1 (Prototype - Sequential):
└── Task 1: Extract analytics_manager.go (proof of concept)

Wave 2 (After Prototype Success - Parallel):
├── Task 2: Extract home_keys.go
├── Task 3: Extract home_workers.go  
└── Task 4: Extract preview_manager.go

Wave 3 (After Wave 2 - Parallel):
├── Task 5: Extract ai_manager.go
└── Task 6: Extract home_state.go

Wave 4 (Interface & Audit - Parallel):
├── Task 7: Add SessionProvider interface
└── Task 8: Performance audit

Critical Path: Task 0 → Task 1 → (Tasks 2-4) → (Tasks 5-6) → (Tasks 7-8)
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 0 | None | 1 | None |
| 1 | 0 | 2, 3, 4 | None (prototype) |
| 2 | 1 | 5, 6 | 3, 4 |
| 3 | 1 | 5, 6 | 2, 4 |
| 4 | 1 | 5, 6 | 2, 3 |
| 5 | 2, 3, 4 | 7, 8 | 6 |
| 6 | 2, 3, 4 | 7, 8 | 5 |
| 7 | 5, 6 | None | 8 |
| 8 | 5, 6 | None | 7 |

---

## Explicit Exclusions (DO NOT TOUCH)

- `internal/ui/*dialog*.go` - Already well-factored
- `internal/ui/search.go`, `global_search.go` - Separate concerns
- `internal/session/*.go` (except interface addition)
- `cmd/agent-deck/*.go` - Out of scope
- `internal/tmux/`, `internal/mcppool/` - Unrelated
- Test files (unless tests fail and require update)

---

## Discovered Issues Tracking

Create `.sisyphus/discovered-issues.md` during refactoring. Log any "while I'm here" improvements here instead of fixing them. Example:

```markdown
# Discovered Issues (DO NOT FIX DURING REFACTORING)

## Code Quality
- [ ] home.go:1234 - Could simplify this loop
- [ ] home.go:2345 - Dead code detected

## Performance Opportunities  
- [ ] Unnecessary allocation in renderPreview()

## Tech Debt
- [ ] Missing error handling in X
```

---

## TODOs

- [x] 0. Pre-Flight: Validate Estimates and Create Rollback Point

  **What to do**:
  - Validate LOC estimates match reality
  - Create git tag for rollback
  - Create discovered-issues tracking file
  - Verify test command works

  **Must NOT do**:
  - Make any code changes
  - Start extractions

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple validation, no code changes
  - **Skills**: [`git-master`]
    - `git-master`: For tagging

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 0 (prerequisite)
  - **Blocks**: Task 1
  - **Blocked By**: None

  **References**:
  - `internal/ui/home.go` - Source file to validate
  - `internal/ui/home_test.go` - Verify test coverage areas

  **Acceptance Criteria**:

  ```bash
  # Verify LOC
  wc -l internal/ui/home.go | awk '{print $1}'
  # Assert: ~7355 (baseline)

  # Count key-related methods
  grep -c "func (h \*Home).*Key" internal/ui/home.go
  # Assert: >= 5 (handlers to extract)

  # Count worker-related methods
  grep -c "Worker\|Background\|logWorker\|statusWorker" internal/ui/home.go
  # Assert: >= 3 (workers to extract)

  # Verify test command
  make test 2>&1 | tail -3
  # Assert: Contains "ok" or "PASS"

  # Verify git tag created
  git tag | grep pre-decomposition
  # Assert: Shows "pre-decomposition"

  # Verify issues file created
  test -f .sisyphus/discovered-issues.md && echo "exists"
  # Assert: "exists"
  ```

  **Commit**: YES
  - Message: `chore: prepare for home.go decomposition`
  - Files: `.sisyphus/discovered-issues.md`
  - Pre-commit: N/A

---

- [x] 1. Prototype: Extract analytics_manager.go (~300 LOC)

  **What to do**:
  - Create `internal/ui/analytics_manager.go`
  - Move analytics-related fields from Home to AnalyticsManager struct
  - Move analytics-related methods to new file
  - Embed AnalyticsManager in Home struct
  - Verify build/test/lint after move

  **Must NOT do**:
  - Change any method signatures
  - Modify method behavior
  - Fix any discovered issues (log to discovered-issues.md)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Careful refactoring requiring attention to detail
  - **Skills**: []
    - No special skills needed, standard Go refactoring

  **Parallelization**:
  - **Can Run In Parallel**: NO (prototype to validate approach)
  - **Parallel Group**: Wave 1 (sequential)
  - **Blocks**: Tasks 2, 3, 4
  - **Blocked By**: Task 0

  **References**:
  - `internal/ui/home.go:153-161` - Analytics cache fields to move
  - `internal/ui/home.go:1357-1425` - `getAnalyticsForSession()`, `fetchAnalytics()` methods
  - `internal/ui/analytics_panel.go` - Existing analytics UI (don't touch, reference patterns)
  - `internal/session/analytics.go` - SessionAnalytics type definition

  **Methods to Extract**:
  ```go
  // Fields (from Home struct)
  currentAnalytics       *session.SessionAnalytics
  currentGeminiAnalytics *session.GeminiSessionAnalytics
  analyticsSessionID     string
  analyticsFetchingID    string
  analyticsCache         map[string]*session.SessionAnalytics
  geminiAnalyticsCache   map[string]*session.GeminiSessionAnalytics
  analyticsCacheTime     map[string]time.Time

  // Methods to move
  func (h *Home) getAnalyticsForSession(inst *session.Instance) *session.SessionAnalytics
  func (h *Home) fetchAnalytics(inst *session.Instance) tea.Cmd
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists
  test -f internal/ui/analytics_manager.go && echo "exists"
  # Assert: "exists"

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | grep -E "ok|PASS" | head -5
  # Assert: Contains "ok" for internal/ui

  # Verify lint passes
  make lint 2>&1 | grep -c "error" || echo "0"
  # Assert: 0

  # Verify home.go reduced (small reduction expected)
  wc -l internal/ui/home.go | awk '{print $1}'
  # Assert: < 7100 (reduced by ~300)
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract analytics subsystem to analytics_manager.go`
  - Files: `internal/ui/analytics_manager.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 2. Extract home_keys.go (~1,500 LOC)

  **What to do**:
  - Create `internal/ui/home_keys.go`
  - Move all `handle*Key` methods to new file
  - Keep methods as receivers on `*Home` (same package)
  - Verify build/test/lint after move

  **Must NOT do**:
  - Change any method signatures
  - Consolidate or refactor handlers
  - Create new abstractions

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Large extraction, careful verification needed
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 3, 4)
  - **Blocks**: Tasks 5, 6
  - **Blocked By**: Task 1

  **References**:
  - `internal/ui/home.go:2857-4095` - All key handler methods
  - `internal/ui/home.go:3103` - `handleMainKey` (largest handler)

  **Methods to Extract**:
  ```go
  func (h *Home) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleGlobalSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleGlobalSearchSelection(result *GlobalSearchResult) tea.Cmd
  func (h *Home) handleNewDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleConfirmDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleMCPDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleGroupDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  func (h *Home) handleForkDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
  // Plus any other handle*Key methods
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists with content
  wc -l internal/ui/home_keys.go | awk '{print $1}'
  # Assert: >= 1000 (substantial content)

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | grep "internal/ui" | head -3
  # Assert: Contains "ok"

  # Verify home.go significantly reduced
  wc -l internal/ui/home.go | awk '{print $1}'
  # Assert: < 5800 (reduced by ~1500 from previous)
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract key handlers to home_keys.go`
  - Files: `internal/ui/home_keys.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 3. Extract home_workers.go (~500 LOC)

  **What to do**:
  - Create `internal/ui/home_workers.go`
  - Move background worker methods and related types
  - Ensure mutexes used by workers move WITH the methods
  - Verify build/test/lint after move

  **Must NOT do**:
  - Change worker behavior
  - Modify tick intervals or timing
  - Split worker logic from its synchronization primitives

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Thread-safety critical, careful verification
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 2, 4)
  - **Blocks**: Tasks 5, 6
  - **Blocked By**: Task 1

  **References**:
  - `internal/ui/home.go:1536-1662` - Worker methods
  - `internal/ui/home.go:1147` - `tick()` method
  - `internal/ui/AGENTS.md` - Background worker documentation

  **Methods to Extract**:
  ```go
  func (h *Home) statusWorker()
  func (h *Home) startLogWorkers()
  func (h *Home) logWorker()
  func (h *Home) backgroundStatusUpdate()
  func (h *Home) syncNotificationsBackground()
  func (h *Home) tick() tea.Cmd
  func (h *Home) triggerStatusUpdate()
  func (h *Home) processStatusUpdate(req statusUpdateRequest)
  // Plus statusUpdateRequest type if defined locally
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists
  wc -l internal/ui/home_workers.go | awk '{print $1}'
  # Assert: >= 300

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | tail -5
  # Assert: Contains "ok" or "PASS"
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract background workers to home_workers.go`
  - Files: `internal/ui/home_workers.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 4. Extract preview_manager.go (~400 LOC)

  **What to do**:
  - Create `internal/ui/preview_manager.go`
  - Move preview-related fields and methods
  - Ensure previewCacheMu moves WITH preview methods
  - Verify build/test/lint after move

  **Must NOT do**:
  - Change debounce timing
  - Modify cache TTL logic
  - Touch existing preview.go (different concern - rendering)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Cache and mutex handling requires care
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 2, 3)
  - **Blocks**: Tasks 5, 6
  - **Blocked By**: Task 1

  **References**:
  - `internal/ui/home.go:176-190` - Preview cache fields
  - `internal/ui/home.go:1302-1356` - Preview fetch methods
  - `internal/ui/preview.go` - Preview rendering (don't touch, separate concern)

  **Methods to Extract**:
  ```go
  // Fields
  previewCache      map[string]string
  previewCacheTime  map[string]time.Time
  previewCacheMu    sync.RWMutex
  previewFetchingID string
  pendingPreviewID  string
  pendingPreviewTimer *time.Timer

  // Methods
  func (h *Home) fetchPreview(inst *session.Instance) tea.Cmd
  func (h *Home) fetchPreviewDebounced(sessionID string) tea.Cmd
  func (h *Home) invalidatePreviewCache(sessionID string)
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists
  wc -l internal/ui/preview_manager.go | awk '{print $1}'
  # Assert: >= 200

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | tail -5
  # Assert: Contains "ok" or "PASS"
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract preview subsystem to preview_manager.go`
  - Files: `internal/ui/preview_manager.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 5. Extract ai_manager.go (~400 LOC)

  **What to do**:
  - Create `internal/ui/ai_manager.go`
  - Move AI-related fields and methods
  - Keep aiProvider and related functionality together
  - Verify build/test/lint after move

  **Must NOT do**:
  - Touch ai_chat.go (separate UI component)
  - Modify AI provider initialization
  - Change summary generation logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: AI subsystem extraction for feature prep
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 6)
  - **Blocks**: Tasks 7, 8
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `internal/ui/home.go:151` - aiProvider field
  - `internal/ui/home.go:1426-1492` - `generateAISummary()` method
  - `internal/ui/ai_chat.go` - AI chat panel (don't touch)
  - `internal/ai/provider.go` - AI provider interface

  **Methods to Extract**:
  ```go
  // Fields
  aiProvider ai.AIProvider
  aiChatPanel *AIChatPanel

  // Methods  
  func (h *Home) generateAISummary(inst *session.Instance) tea.Cmd
  // Plus any AI-related helper methods
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists
  wc -l internal/ui/ai_manager.go | awk '{print $1}'
  # Assert: >= 150

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | tail -5
  # Assert: Contains "ok" or "PASS"
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract AI subsystem to ai_manager.go`
  - Files: `internal/ui/ai_manager.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 6. Extract home_state.go (~300 LOC)

  **What to do**:
  - Create `internal/ui/home_state.go`
  - Move state management and cursor-related methods
  - Move undo stack logic
  - Verify build/test/lint after move

  **Must NOT do**:
  - Change state mutation patterns
  - Modify viewport calculations
  - Touch flatItems rebuilding logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: State management critical for correctness
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 5)
  - **Blocks**: Tasks 7, 8
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `internal/ui/home.go:618-710` - State preservation/restoration
  - `internal/ui/home.go:775-885` - Viewport sync, notifications
  - `internal/ui/home.go:1511-1535` - Undo stack

  **Methods to Extract**:
  ```go
  func (h *Home) preserveState() reloadState
  func (h *Home) restoreState(state reloadState)
  func (h *Home) rebuildFlatItems()
  func (h *Home) syncViewport()
  func (h *Home) pushUndoStack(inst *session.Instance)
  func (h *Home) getSelectedSession() *session.Instance
  func (h *Home) getInstanceByID(id string) *session.Instance
  func (h *Home) jumpToSession(inst *session.Instance)
  func (h *Home) jumpToRootGroup(n int)
  // Plus reloadState type
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify new file exists
  wc -l internal/ui/home_state.go | awk '{print $1}'
  # Assert: >= 200

  # Verify build succeeds
  go build ./...
  # Assert: exit code 0

  # Verify tests pass
  make test 2>&1 | tail -5
  # Assert: Contains "ok" or "PASS"
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract state management to home_state.go`
  - Files: `internal/ui/home_state.go`, `internal/ui/home.go`
  - Pre-commit: `go build ./... && make test`

---

- [x] 7. Add SessionProvider Interface

  **What to do**:
  - Create `internal/session/interfaces.go`
  - Define `SessionProvider` interface with key Instance methods
  - Verify Instance implicitly implements interface
  - Document interface purpose

  **Must NOT do**:
  - Modify Instance struct
  - Add more interfaces (only SessionProvider)
  - Change any existing code to use the interface yet

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small, focused interface addition
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Task 8)
  - **Blocks**: None
  - **Blocked By**: Tasks 5, 6

  **References**:
  - `internal/session/instance.go:34-100` - Instance struct and key methods
  - `internal/ui/home.go` - How Instance is used in UI layer

  **Interface Design**:
  ```go
  // SessionProvider defines the interface for session operations
  // used by the UI layer. This enables testing without real sessions.
  type SessionProvider interface {
      GetID() string
      GetName() string
      GetStatus() Status
      GetPath() string
      GetToolType() string
      IsRunning() bool
      Start() error
      Stop() error
  }
  
  // Verify Instance implements SessionProvider
  var _ SessionProvider = (*Instance)(nil)
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify interface file exists
  test -f internal/session/interfaces.go && echo "exists"
  # Assert: "exists"

  # Verify interface is defined
  grep -c "type SessionProvider interface" internal/session/interfaces.go
  # Assert: 1

  # Verify Instance implements it (build succeeds)
  go build ./...
  # Assert: exit code 0

  # Verify no "does not implement" errors
  go build ./... 2>&1 | grep -c "does not implement" || echo "0"
  # Assert: 0
  ```

  **Commit**: YES
  - Message: `refactor(session): add SessionProvider interface for testing`
  - Files: `internal/session/interfaces.go`
  - Pre-commit: `go build ./...`

---

- [x] 8. Performance Audit: Startup Profiling

  **What to do**:
  - Profile startup time with `go test -bench`
  - Identify slow initialization paths
  - Document findings in `docs/performance-audit.md`
  - List actionable optimizations (DO NOT implement)

  **Must NOT do**:
  - Implement any optimizations (document only)
  - Modify any code
  - Add benchmarks to existing test files

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Audit and documentation only
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Task 7)
  - **Blocks**: None
  - **Blocked By**: Tasks 5, 6

  **References**:
  - `internal/ui/home.go:415-617` - NewHome initialization
  - `internal/ui/home.go:1077-1100` - Init() method
  - `internal/session/storage.go` - LoadWithGroups

  **Profiling Commands**:
  ```bash
  # CPU profile during startup
  go test -cpuprofile=cpu.prof -bench=. ./internal/ui/... -run=^$ -benchtime=10s

  # Trace startup
  go test -trace=trace.out ./internal/ui/... -run=TestNewHome

  # Analyze
  go tool pprof cpu.prof
  go tool trace trace.out
  ```

  **Acceptance Criteria**:

  ```bash
  # Verify audit document exists
  test -f docs/performance-audit.md && echo "exists"
  # Assert: "exists"

  # Verify document has content
  wc -l docs/performance-audit.md | awk '{print $1}'
  # Assert: >= 50 (substantial findings)

  # Verify document structure
  grep -c "## " docs/performance-audit.md
  # Assert: >= 3 (multiple sections)
  ```

  **Commit**: YES
  - Message: `docs: add performance audit for startup optimization`
  - Files: `docs/performance-audit.md`
  - Pre-commit: N/A

---

## Commit Strategy

| After Task | Message | Files | Verification |
|------------|---------|-------|--------------|
| 0 | `chore: prepare for home.go decomposition` | `.sisyphus/discovered-issues.md` | git tag exists |
| 1 | `refactor(ui): extract analytics subsystem to analytics_manager.go` | `analytics_manager.go`, `home.go` | build + test |
| 2 | `refactor(ui): extract key handlers to home_keys.go` | `home_keys.go`, `home.go` | build + test |
| 3 | `refactor(ui): extract background workers to home_workers.go` | `home_workers.go`, `home.go` | build + test |
| 4 | `refactor(ui): extract preview subsystem to preview_manager.go` | `preview_manager.go`, `home.go` | build + test |
| 5 | `refactor(ui): extract AI subsystem to ai_manager.go` | `ai_manager.go`, `home.go` | build + test |
| 6 | `refactor(ui): extract state management to home_state.go` | `home_state.go`, `home.go` | build + test |
| 7 | `refactor(session): add SessionProvider interface for testing` | `interfaces.go` | build |
| 8 | `docs: add performance audit for startup optimization` | `performance-audit.md` | file exists |

---

## Success Criteria

### Verification Commands
```bash
# Final home.go size
wc -l internal/ui/home.go | awk '{print $1}'
# Expected: < 4000

# All new files exist
ls internal/ui/home_keys.go internal/ui/home_workers.go internal/ui/home_state.go internal/ui/preview_manager.go internal/ui/analytics_manager.go internal/ui/ai_manager.go 2>/dev/null | wc -l
# Expected: 6

# Interface exists
grep -l "SessionProvider" internal/session/interfaces.go
# Expected: internal/session/interfaces.go

# All tests pass
make test 2>&1 | tail -1
# Expected: Contains "ok" or exit 0

# Audit document exists
test -f docs/performance-audit.md && echo "exists"
# Expected: exists
```

### Final Checklist
- [ ] `home.go` < 4,000 LOC (ACTUAL: 5,309 - target not met, see recommendations)
- [x] 6 new extraction files created
- [x] SessionProvider interface added
- [x] Performance audit documented
- [ ] All tests pass (internal/ui passes, pre-existing failures in other packages)
- [x] No "discovered issues" fixed (logged to separate file)
- [x] Each extraction has its own commit
