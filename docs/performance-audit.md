# Performance Audit: Startup Profiling

**Date:** 2026-01-31  
**Profiling Tool:** Go pprof (CPU, memory) + runtime/trace  
**Test Subject:** `NewHome()` initialization (100 iterations)  
**Total Profiling Duration:** 5.78s CPU time, 21.5ms per initialization

## Executive Summary

The `NewHome()` initialization path exhibits **three major bottlenecks** that collectively account for ~71% of CPU time and ~72% of heap allocations:

1. **Global Search Index Loading (71.26% CPU, 71.75% heap)** - Walks entire Claude conversation directory, parses JSONL files
2. **Log File Maintenance (10.65% CPU, 16.93% heap)** - Truncates large log files during startup
3. **JSON Unmarshaling (44.10% CPU)** - Decoding Claude conversation history

These are **initialization-time costs** that occur once per app startup. The profile reveals that most time is spent in **I/O and memory allocation** rather than algorithmic complexity.

## Methodology

### Profiling Setup
- **Test File:** `internal/ui/profile_test.go`
- **Test Functions:**
  - `TestNewHomeCPU()` - CPU profiling with 100 iterations
  - `TestNewHomeTrace()` - Runtime trace for goroutine analysis
  - `BenchmarkNewHome()` - Benchmark harness

### Execution
```bash
go test -run=TestNewHomeCPU ./internal/ui/... -v
go test -run=TestNewHomeTrace ./internal/ui/... -v
```

### Analysis Tools
```bash
go tool pprof -top cpu.prof
go tool pprof -top -cum cpu.prof
go tool pprof -top mem.prof
```

## Key Findings

### 1. Global Search Index Initialization (CRITICAL)

**Impact:** 71.26% of CPU time, 71.75% of heap allocations (141.53 MB)

**Call Stack:**
```
NewHomeWithProfileAndMode
  → NewGlobalSearchIndex()
    → GlobalSearchIndex.initialLoad()
      → filepath.WalkDir() [12.25s cumulative]
        → parseClaudeJSONL() [10.61s cumulative]
          → json.Unmarshal() [9.26s cumulative]
            → json.(*decodeState).object() [7.20s cumulative]
```

**Root Cause:** The global search feature walks the entire `~/.claude/projects/` directory tree and parses every JSONL conversation file. This is **synchronous during initialization**, blocking the UI from rendering.

**Specific Costs:**
- `filepath.WalkDir()`: 12.25s (directory traversal)
- `parseClaudeJSONL()`: 10.61s (JSON parsing)
- `json.Unmarshal()`: 9.26s (deserialization)
- `encoding/json.(*decodeState).object()`: 7.20s (object decoding)
- Memory: 135.73 MB from `os.readFileContents` (reading all JSONL files into memory)

**Observation:** The profile shows `initialLoad.func1` is a goroutine, but it's still blocking the main initialization path. The `NewGlobalSearchIndex()` call waits for the index to be ready before returning.

### 2. Log File Maintenance (HIGH)

**Impact:** 10.65% of CPU time, 16.93% of heap allocations (33.41 MB)

**Call Stack:**
```
NewHomeWithProfileAndMode
  → go func() { RunLogMaintenance() }
    → TruncateLargeLogFiles() [1.33s cumulative]
      → TruncateLogFile() [1.24s cumulative]
        → strings.(*Builder).WriteString() [12.40 MB]
        → strings.(*Builder).grow() [12.11 MB]
```

**Root Cause:** Log maintenance runs in a background goroutine but still allocates significant memory. The `TruncateLogFile()` function reads entire log files into memory, processes them with string builders, and writes them back.

**Specific Costs:**
- `TruncateLargeLogFiles()`: 1.33s (iterating log files)
- `TruncateLogFile()`: 1.24s (per-file truncation)
- String builder allocations: 24.51 MB (12.40 + 12.11 MB)
- File I/O: `os.ReadFile()` 1.65s cumulative

**Observation:** This runs in the background but still impacts startup time perception and memory footprint.

### 3. JSON Unmarshaling (HIGH)

**Impact:** 44.10% of CPU time (cumulative through all JSON operations)

**Breakdown:**
- `json.(*decodeState).unmarshal()`: 7.58s
- `json.(*decodeState).object()`: 7.20s
- `json.(*decodeState).array()`: 4.02s
- `encoding/json.checkValid()`: 1.10s
- `encoding/json.unquoteBytes()`: 0.88s

**Root Cause:** Parsing large JSONL files with complex nested structures. The JSON decoder allocates new objects for each field, and the cumulative effect is significant.

**Observation:** This is inherent to the data format. Optimization would require either:
- Streaming JSON parsing (not standard library)
- Binary format (protobuf, msgpack)
- Lazy deserialization

### 4. Memory Allocation Hotspots

**Top Allocators (by heap size):**
1. `os.readFileContents`: 135.73 MB (68.80%) - Reading JSONL files
2. `TruncateLogFile`: 33.41 MB (16.93%) - String building for log truncation
3. `parseClaudeJSONL`: 22.51 MB (11.41%) - JSON parsing allocations
4. Dialog initialization: 4.07 MB (2.06%) - NewNewDialog, NewWatchDialog, etc.

**Total Heap Used:** 197.27 MB (100 iterations of NewHome)

### 5. Initialization Sequence Timing

**Single Initialization (from trace):** 21.544 ms

**Breakdown (estimated from profile):**
- Global search index: ~15 ms (71% of 21.5 ms)
- Dialog initialization: ~2 ms (10%)
- Storage/config loading: ~2 ms (10%)
- Other: ~2.5 ms (9%)

**Observation:** Global search dominates the critical path. The 21.5 ms is acceptable for app startup, but the blocking nature means the splash screen is visible for this duration.

## Actionable Optimizations (DO NOT IMPLEMENT)

### Priority 1: Defer Global Search Index Loading

**Potential Savings:** 71% of startup time (15 ms), 141.53 MB heap

**Approach:**
- Load global search index in background after UI renders
- Show "Global search initializing..." status
- Disable global search until index is ready
- Cache index to disk to avoid re-parsing on subsequent startups

**Implementation Complexity:** Medium
- Requires async initialization pattern
- Need to handle "not ready" state in search UI
- Disk caching adds complexity but huge benefit

### Priority 2: Stream-Based JSON Parsing for JSONL

**Potential Savings:** 44% of JSON time (4-5 ms), 22.51 MB heap

**Approach:**
- Use `json.Decoder` (streaming) instead of `json.Unmarshal` (buffered)
- Parse one conversation at a time instead of loading entire file
- Reduces memory footprint from 135 MB to ~10 MB

**Implementation Complexity:** Low
- Drop-in replacement in `parseClaudeJSONL()`
- No API changes needed

### Priority 3: Lazy Log File Truncation

**Potential Savings:** 10.65% of startup time (2.3 ms), 33.41 MB heap

**Approach:**
- Move log maintenance to background task (already in goroutine, but still allocates)
- Implement incremental truncation (process one file per tick)
- Use buffered I/O instead of loading entire files

**Implementation Complexity:** Medium
- Requires incremental processing logic
- Need to track truncation state across ticks

### Priority 4: Dialog Lazy Initialization

**Potential Savings:** 2% of startup time (0.4 ms), 4.07 MB heap

**Approach:**
- Create dialogs on-demand when first shown
- Pre-create only essential dialogs (NewDialog, ForkDialog)
- Defer optional dialogs (SettingsPanel, AnalyticsPanel)

**Implementation Complexity:** Low
- Requires nil-check guards in Update()
- Minimal risk of regression

### Priority 5: Reduce JSON Allocations

**Potential Savings:** 5-10% of JSON time (0.5-1 ms)

**Approach:**
- Use `json.RawMessage` for large nested fields
- Implement custom `UnmarshalJSON()` with object pooling
- Pre-allocate slices with capacity hints

**Implementation Complexity:** High
- Requires deep understanding of JSON decoder internals
- Risk of subtle bugs with custom unmarshaling

## Performance Characteristics

### Scaling Behavior

**Hypothesis:** Startup time scales linearly with number of Claude conversations.

**Evidence:**
- Global search walks entire directory tree: O(n) where n = conversation count
- JSON parsing is O(n) in file size
- Log maintenance is O(m) where m = log file count

**Implication:** Users with 1000+ conversations will experience 5-10x slower startup.

### Memory Behavior

**Peak Memory:** 197.27 MB for 100 iterations (1.97 MB per iteration)

**Breakdown:**
- Global search: 141.53 MB (71.75%)
- Log maintenance: 33.41 MB (16.93%)
- Dialogs + other: 22.33 MB (11.32%)

**Implication:** Memory is released after initialization, but peak usage is significant.

## Recommendations for Future Work

### Short-term (1-2 sprints)
1. Implement streaming JSON parsing for JSONL files
2. Defer global search index loading to background
3. Add startup time metrics to telemetry

### Medium-term (1-2 months)
1. Implement disk caching for global search index
2. Incremental log file truncation
3. Lazy dialog initialization

### Long-term (3+ months)
1. Consider binary format for conversation storage (protobuf)
2. Implement conversation indexing service (separate process)
3. Profile other initialization paths (MCP discovery, tmux status)

## Appendix: Raw Data

### CPU Profile (Top 20 Functions)

```
File: ui.test
Type: cpu
Duration: 5.78s, Total samples = 17.19s (297.36%)

      flat  flat%   cum   cum%
     5.82s 33.86%  5.82s 33.86%  runtime.memclrNoHeapPointers
     2.85s 16.58%  2.85s 16.58%  runtime.memmove
     0.95s  5.53%  0.97s  5.64%  syscall.syscall
     0.77s  4.48%  0.77s  4.48%  runtime.nextFreeFast
     0.75s  4.36%  0.77s  4.48%  encoding/json.stateInString
     0.60s  3.49%  0.60s  3.49%  atomic.(*UnsafePointer).StoreNoWB
     0.57s  3.32%  0.57s  3.32%  runtime.pthread_cond_wait
     0.43s  2.50%  0.44s  2.56%  syscall.syscall6
     0.37s  2.15%  0.37s  2.15%  runtime.madvise
     0.23s  1.34%  0.70s  4.07%  encoding/json.(*decodeState).skip
     0.23s  1.34%  1.10s  6.40%  encoding/json.checkValid
     0.13s  0.76%  0.13s  0.76%  runtime.kevent
     0.12s  0.70%  0.12s  0.70%  encoding/json.stateInStringEsc
     0.12s  0.70%  0.12s  0.70%  runtime.(*mspan).init
     0.11s  0.64%  0.11s  0.64%  runtime.acquirem
     0.11s  0.64%  0.16s  0.93%  runtime.mallocgcTiny
     0.11s  0.64%  0.11s  0.64%  runtime.pthread_kill
     0.10s  0.58%  2.51s 14.60%  runtime.mallocgcSmallScanNoHeader
     0.10s  0.58%  0.10s  0.58%  syscall.syscallPtr
     0.09s  0.52%  0.10s  0.58%  encoding/json.stateEndValue
```

### Memory Profile (Top 20 Allocators)

```
File: ui.test
Type: inuse_space
Time: 2026-01-31 00:56:35 PST
Total: 197.27 MB

  135.73MB 68.80%  os.readFileContents
   16.70MB  8.47%  github.com/asheshgoplani/agent-deck/internal/tmux.TruncateLogFile
   12.40MB  6.29%  strings.(*Builder).WriteString
   12.11MB  6.14%  strings.(*Builder).grow
    3.06MB  1.55%  github.com/asheshgoplani/agent-deck/internal/ui.NewNewDialog
    3.06MB  1.55%  github.com/asheshgoplani/agent-deck/internal/ui.NewWatchDialog
    2.50MB  1.27%  runtime.allocm
    1.52MB  0.77%  github.com/asheshgoplani/agent-deck/internal/ui.NewForkDialog
    1.16MB  0.59%  runtime/pprof.StartCPUProfile
    1.01MB  0.51%  github.com/asheshgoplani/agent-deck/internal/ui.NewClaudeOptionsPanel
```

### Profiling Commands Used

```bash
# CPU profiling
go test -run=TestNewHomeCPU ./internal/ui/... -v

# Trace profiling
go test -run=TestNewHomeTrace ./internal/ui/... -v

# Analysis
go tool pprof -top cpu.prof
go tool pprof -top -cum cpu.prof
go tool pprof -top mem.prof
go tool pprof -list=NewHome cpu.prof
```

### Test Environment

- **Go Version:** 1.24+
- **Platform:** macOS
- **Test Profile:** `_test` (isolated storage)
- **Iterations:** 100 (CPU), 1 (trace)
- **Date:** 2026-01-31

## Conclusion

The startup performance is **acceptable for a TUI application** (21.5 ms), but there are clear optimization opportunities that could reduce it to **5-10 ms** with moderate effort. The primary bottleneck is **global search index loading**, which should be deferred to background initialization. Secondary optimizations in JSON parsing and log maintenance could provide additional 2-3 ms improvements.

The profile data provides a clear roadmap for future optimization work without requiring code changes at this time.
