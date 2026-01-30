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
