# Thread-Safety Primitives in internal/ui/home.go

## Analysis Date
2026-01-30

## Summary
Mapped all synchronization primitives (mutexes, atomics, channels) in the Home struct and their usage patterns. Key finding: **Mutexes are tightly coupled to specific methods and cannot be safely extracted without careful refactoring**.

---

## Field Declarations (Lines 116-282)

### Mutexes (RWMutex)

| Field | Line | Purpose | Protected Data | Scope |
|-------|------|---------|-----------------|-------|
| `instancesMu` | 127 | Protects instances slice | `instances`, `instanceByID` | Background worker + main goroutine |
| `previewCacheMu` | 179 | Protects preview cache | `previewCache`, `previewCacheTime`, `previewFetchingID` | Async preview fetching |
| `logActivityMu` | 207 | Protects log activity map | `lastLogActivity` | Log watcher callback |

### Mutexes (Mutex)

| Field | Line | Purpose | Protected Data | Scope |
|-------|------|---------|-----------------|-------|
| `reloadMu` | 174 | Protects reload state | `reloadVersion`, `isReloading` | Storage watcher + main |
| `previewDebounceMu` | 185 | Protects debounce state | `pendingPreviewID` | Navigation + debounce timer |
| `boundKeysMu` | 261 | Protects key bindings | `boundKeys` | Background worker + main |
| `lastBarTextMu` | 263 | Protects notification bar text | `lastBarText` | Background worker + main |
| `lastNotifSwitchMu` | 278 | Protects notification switch tracking | `lastNotifSwitchID` | Background worker + main |

### Atomics

| Field | Line | Type | Purpose |
|-------|------|------|---------|
| `isAttaching` | 165 | `atomic.Bool` | Prevents View() output during attach |
| `statusUpdateIndex` | 190 | `atomic.Int32` | Round-robin status update position |
| `valid` (in cachedStatusCounts) | 250 | `atomic.Bool` | Cache validity flag |

### Channels

| Field | Line | Type | Purpose | Buffer |
|-------|------|------|---------|--------|
| `statusTrigger` | 194 | `chan statusUpdateRequest` | Triggers background status update | 1 |
| `statusWorkerDone` | 195 | `chan struct{}` | Signals worker has stopped | 0 |
| `logUpdateChan` | 199 | `chan *session.Instance` | Buffers log-driven status updates | 100 |

---

## Critical Usage Patterns

### Pattern 1: instancesMu - Most Heavily Used (23 lock sites)

**Lock Sites:**
- Line 504-528: Log watcher callback (RLock/Unlock pair)
- Line 921-931: getAttachedSessionID (RLock/Unlock pair)
- Line 936-938: syncNotifications (RLock/Unlock pair)
- Line 956-957: getAttachedSessionID (RLock/defer RUnlock)
- Line 1617-1624: backgroundStatusUpdate (RLock/Unlock pair)
- Line 1697-1713: syncNotificationsBackground (RLock/Unlock pair)
- Line 1774-1793: updateKeyBindings (RLock/Unlock pair)
- Line 1864-1871: processStatusUpdate (RLock/Unlock pair)
- Line 2006-2027: loadSessionsMsg handler (Lock/Unlock pair)
- Line 2097-2102: sessionCreatedMsg handler (Lock/Unlock pair)
- Line 2151-2157: sessionForkedMsg handler (Lock/Unlock pair)
- Line 2205-2214: sessionDeletedMsg handler (Lock/Unlock pair)
- Line 2252-2256: sessionRestoredMsg handler (Lock/Unlock pair)
- Line 2395-2397: handleGlobalSearchSelection (RLock/Unlock pair)
- Line 2466-2468: statusUpdateMsg handler (RLock/Unlock pair)
- Line 2513-2515: previewDebounceMsg handler (RLock/Unlock pair)
- Line 2639-2648: aiSummaryMsg handler (Lock/Unlock pair)
- Line 2739-2741: tickMsg handler (RLock/Unlock pair)
- Line 3740-3742: getOtherActiveSessions (RLock/Unlock pair)
- Line 3818-3820: handleConfirmDialogKey (Lock/Unlock pair)
- Line 4004-4006: deleteSession (Lock/Unlock pair)
- Line 4019-4021: deleteSession (Lock/Unlock pair)
- Line 4135-4139: getInstanceByID (RLock/Unlock pair)
- Line 4172-4173: getInstanceByID (RLock/defer RUnlock)
- Line 4477-4481: saveInstancesWithForce (Lock/Unlock pair)

**Methods That MUST Move Together:**
- `backgroundStatusUpdate()` (line 1606) - reads instances
- `syncNotificationsBackground()` (line 1662) - reads instances
- `processStatusUpdate()` (line 1855) - reads instances
- `updateKeyBindings()` (line 1761) - reads instances
- `getAttachedSessionID()` (line 950) - reads instances
- `getInstanceByID()` (line 1506) - reads instanceByID map
- `getOtherActiveSessions()` - reads instances

**Extraction Risk:** **CRITICAL** - These methods form a cohesive unit for background status updates. Separating them would require passing instances as parameters or creating a separate data structure.

---

### Pattern 2: previewCacheMu - Async Preview Fetching (8 lock sites)

**Lock Sites:**
- Line 1156-1159: invalidatePreviewCache (Lock/Unlock)
- Line 1264-1266: hasActiveAnimation (RLock/Unlock)
- Line 1323-1325: fetchPreviewDebounced (Lock/Unlock)
- Line 2071-2073: loadSessionsMsg handler (Lock/Unlock)
- Line 2521-2526: previewDebounceMsg handler (Lock/Unlock)
- Line 2593-2599: previewFetchedMsg handler (Lock/Unlock)
- Line 2743-2751: tickMsg handler (Lock/Unlock)
- Line 6863-6865: renderPreviewPane (RLock/Unlock)
- Line 6907-6909: renderPreviewPane (RLock/Unlock)

**Methods That MUST Move Together:**
- `invalidatePreviewCache()` (line 1155)
- `fetchPreviewDebounced()` (line 1320)
- `hasActiveAnimation()` (line 1206)

**Extraction Risk:** **HIGH** - These methods are tightly coupled to preview rendering and caching logic. Cannot be separated without passing cache state as parameters.

---

### Pattern 3: reloadMu - Reload State Protection (6 lock sites)

**Lock Sites:**
- Line 1966-1968: loadSessionsMsg handler (Lock/Unlock)
- Line 2415-2418: storageChangedMsg handler (Lock/Unlock)
- Line 2484-2486: statusUpdateMsg handler (Lock/Unlock)
- Line 4389-4391: saveInstancesWithForce (Lock/Unlock)

**Methods That MUST Move Together:**
- Reload state checks in: `loadSessionsMsg`, `sessionCreatedMsg`, `sessionForkedMsg`, `sessionDeletedMsg`, `sessionRestoredMsg`, `statusUpdateMsg`

**Extraction Risk:** **MEDIUM** - Reload state is checked in multiple message handlers. Could be extracted if reload state is passed as a parameter or stored in a separate component.

---

### Pattern 4: Background Worker Mutexes (boundKeysMu, lastBarTextMu, lastNotifSwitchMu)

**Lock Sites:**
- `boundKeysMu`: Lines 1007, 1012, 1796, 1812 (updateKeyBindings)
- `lastBarTextMu`: Lines 976, 979, 990, 1734, 1737, 1750 (updateTmuxNotifications, syncNotificationsBackground)
- `lastNotifSwitchMu`: Lines 1687, 1689, 2449, 2452 (syncNotificationsBackground, statusUpdateMsg)

**Methods That MUST Move Together:**
- `updateKeyBindings()` (line 1761) - uses boundKeysMu
- `updateTmuxNotifications()` (line 971) - uses lastBarTextMu
- `syncNotificationsBackground()` (line 1662) - uses lastBarTextMu, lastNotifSwitchMu

**Extraction Risk:** **MEDIUM-HIGH** - These are called from both foreground and background goroutines. Thread-safety is critical. Cannot be extracted without careful synchronization redesign.

---

### Pattern 5: Atomic Operations (3 sites)

**isAttaching (atomic.Bool):**
- Line 1686: Load in syncNotificationsBackground
- Line 2440: Store in statusUpdateMsg handler
- Line 4660: Load in View() method

**statusUpdateIndex (atomic.Int32):**
- Line 1902: Load in processStatusUpdate
- Line 1926: Store in processStatusUpdate

**cachedStatusCounts.valid (atomic.Bool):**
- Line 2029: Store in loadSessionsMsg
- Line 2104: Store in sessionCreatedMsg
- Line 2159: Store in sessionForkedMsg
- Line 2222: Store in sessionDeletedMsg
- Line 2257: Store in sessionRestoredMsg
- Line 4501: Load in getStatusCounts
- Line 4508-4521: RLock instances after atomic check

**Extraction Risk:** **LOW** - Atomics are self-contained and can be extracted independently.

---

### Pattern 6: Channels (statusTrigger, statusWorkerDone, logUpdateChan)

**statusTrigger Usage:**
- Line 481: Initialization (buffered, size 1)
- Line 1554: Receive in statusWorker goroutine
- Line 1839: Send in triggerStatusUpdate

**statusWorkerDone Usage:**
- Line 482: Initialization
- Line 1537: Close in statusWorker
- Line 3875: Receive in performFinalShutdown

**logUpdateChan Usage:**
- Line 483: Initialization (buffered, size 100)
- Line 521: Send in log watcher callback
- Line 586: Receive in logWorker goroutine

**Extraction Risk:** **MEDIUM** - Channels are tightly coupled to goroutine lifecycle. Cannot be extracted without careful redesign of worker coordination.

---

## Mutex Dependencies & Extraction Constraints

### Cannot Extract Without Refactoring:

1. **instancesMu + background worker methods**
   - `backgroundStatusUpdate()`, `syncNotificationsBackground()`, `processStatusUpdate()`, `updateKeyBindings()`
   - These form a cohesive unit for background status updates
   - Extracting requires: passing instances as parameter or creating a separate data structure

2. **previewCacheMu + preview methods**
   - `invalidatePreviewCache()`, `fetchPreviewDebounced()`, `hasActiveAnimation()`
   - Tightly coupled to preview rendering
   - Extracting requires: passing cache state as parameter

3. **reloadMu + reload state checks**
   - Checked in 6+ message handlers
   - Extracting requires: passing reload state as parameter or creating a reload state manager

4. **Background worker coordination**
   - `boundKeysMu`, `lastBarTextMu`, `lastNotifSwitchMu` are used in methods called from both foreground and background
   - Extracting requires: careful synchronization redesign

### Can Extract With Minimal Refactoring:

1. **Atomic operations** - self-contained, can be extracted independently
2. **Channel-based coordination** - can be extracted if worker lifecycle is preserved

---

## Recommendations for Safe Extraction

### Phase 1: Extract Atomics (Low Risk)
- Extract `isAttaching`, `statusUpdateIndex`, `cachedStatusCounts.valid` to a separate `AtomicState` struct
- No mutex dependencies, minimal refactoring needed

### Phase 2: Extract Channels (Medium Risk)
- Extract `statusTrigger`, `statusWorkerDone`, `logUpdateChan` to a separate `WorkerCoordination` struct
- Requires careful handling of goroutine lifecycle
- Must preserve buffering semantics

### Phase 3: Extract Background Worker Methods (High Risk)
- Create a `BackgroundWorker` struct that owns:
  - `instancesMu`, `boundKeysMu`, `lastBarTextMu`, `lastNotifSwitchMu`
  - Methods: `backgroundStatusUpdate()`, `syncNotificationsBackground()`, `processStatusUpdate()`, `updateKeyBindings()`
- Requires: passing instances reference or creating a shared data structure
- Must maintain thread-safety guarantees

### Phase 4: Extract Preview Methods (High Risk)
- Create a `PreviewManager` struct that owns:
  - `previewCacheMu`
  - Methods: `invalidatePreviewCache()`, `fetchPreviewDebounced()`, `hasActiveAnimation()`
- Requires: passing cache state as parameter or creating a shared cache structure

### Phase 5: Extract Reload State (Medium Risk)
- Create a `ReloadState` struct that owns:
  - `reloadMu`, `reloadVersion`, `isReloading`
- Requires: passing reload state to message handlers or creating a reload manager

---

## Key Insights

1. **Mutexes protect related data**: Each mutex protects a cohesive set of fields. Extracting methods requires extracting the protected data too.

2. **Background workers are tightly coupled**: Methods called from background goroutines (statusWorker, logWorker) are interdependent and cannot be easily separated.

3. **Thread-safety is critical**: The Home struct manages concurrent access from:
   - Main Bubble Tea goroutine (Update, View)
   - Background status worker (statusWorker)
   - Log watcher callback (log file monitoring)
   - Async command handlers (preview fetching, analytics parsing)

4. **Extraction order matters**: Must extract in dependency order:
   - Atomics first (no dependencies)
   - Channels second (minimal dependencies)
   - Background workers third (depends on instances)
   - Preview methods fourth (depends on cache)
   - Reload state last (depends on message handlers)

5. **No "safe" extraction without design changes**: Every mutex extraction requires either:
   - Passing protected data as parameters
   - Creating a separate data structure to hold the data
   - Redesigning the synchronization strategy

