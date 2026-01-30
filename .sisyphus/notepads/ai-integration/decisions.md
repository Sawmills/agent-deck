# Decisions - AI Integration

## Architecture Decisions
- Primary AI: Claude Opus 4.5 (Anthropic)
- Multi-provider support: Anthropic, OpenAI, OpenRouter
- Observation storage: Persistent JSON with FIFO retention
- Watch scope: Flexible (all sessions OR specific sessions per goal)
- Multiple concurrent watch goals: YES (max 10)
- Conversation persistence: NO (v1 is stateless Q&A)
- Automated actions: NO (notify/suggest only)

## Guardrails
- MAX_OBSERVATION_SIZE: 50KB
- MAX_OBSERVATIONS_PER_SESSION: 100 (FIFO eviction)
- MAX_CONCURRENT_GOALS: 10
- AI_REQUEST_TIMEOUT: 30s
- DEFAULT_GOAL_TIMEOUT: 1 hour
