# Discovered Issues (DO NOT FIX DURING REFACTORING)

## Test Failures
- [ ] `make test` reports FAIL in test suite (noted during pre-flight validation)
  - Status: Not investigated (per instructions, only logging)
  - Impact: Baseline test state captured at tag `pre-decomposition`

## Code Quality
- [ ] home.go has 12 Key-related handler methods to extract
- [ ] home.go has 49 Worker/Background/logWorker/statusWorker references to consolidate

## Performance Opportunities
- [ ] MCP socket pooling could be optimized further (currently 85-90% reduction)

## Tech Debt
- [ ] home.go at 7355 LOC needs decomposition into focused modules
- [ ] Consider extracting key handlers into separate file
- [ ] Consider extracting worker logic into separate file

---

## Refactoring Baseline (pre-decomposition tag)

**Captured:** 2026-01-30
**Commit:** e1dd049
**Branch:** feat/Claude-session-forking

### Metrics
- home.go LOC: 7355
- Key handlers: 12
- Worker references: 49
- Test status: FAIL (baseline captured)

### Rollback Point
Tag: `pre-decomposition`
Command: `git checkout pre-decomposition`
