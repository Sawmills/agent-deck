package ui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// AnalyticsManager encapsulates analytics cache state and async fetching logic.
// These fields currently live on Home (for compatibility) but are candidates
// for extraction via embedding in a future refactoring wave.
type AnalyticsManager struct {
	currentAnalytics       *session.SessionAnalytics                  // Current analytics for selected session (Claude)
	currentGeminiAnalytics *session.GeminiSessionAnalytics            // Current analytics for selected session (Gemini)
	analyticsSessionID     string                                     // Session ID for current analytics
	analyticsFetchingID    string                                     // ID currently being fetched (prevents duplicates)
	analyticsCache         map[string]*session.SessionAnalytics       // TTL cache: sessionID -> analytics (Claude)
	geminiAnalyticsCache   map[string]*session.GeminiSessionAnalytics // TTL cache: sessionID -> analytics (Gemini)
	analyticsCacheTime     map[string]time.Time                       // TTL cache: sessionID -> cache timestamp
}

// getAnalyticsForSession returns cached analytics if still valid (within TTL)
// Returns nil if cache miss or expired, triggering async fetch
func (h *Home) getAnalyticsForSession(inst *session.Instance) *session.SessionAnalytics {
	if inst == nil {
		return nil
	}

	// Check cache
	if cached, ok := h.analyticsCache[inst.ID]; ok {
		if time.Since(h.analyticsCacheTime[inst.ID]) < analyticsCacheTTL {
			return cached
		}
	}

	return nil // Will trigger async fetch
}

// fetchAnalytics returns a command that asynchronously parses session analytics
// This keeps View() pure (no blocking I/O) as per Bubble Tea best practices
func (h *Home) fetchAnalytics(inst *session.Instance) tea.Cmd {
	if inst == nil {
		return nil
	}
	sessionID := inst.ID

	if inst.Tool == "claude" {
		claudeSessionID := inst.ClaudeSessionID
		return func() tea.Msg {
			// Get JSONL path for this session
			jsonlPath := inst.GetJSONLPath()
			if jsonlPath == "" {
				// No JSONL path available - return empty analytics
				return analyticsFetchedMsg{
					sessionID: sessionID,
					analytics: nil,
					err:       nil,
				}
			}

			// Parse the JSONL file
			analytics, err := session.ParseSessionJSONL(jsonlPath)
			if err != nil {
				log.Printf("Failed to parse analytics for session %s (claude session %s): %v", sessionID, claudeSessionID, err)
				return analyticsFetchedMsg{
					sessionID: sessionID,
					analytics: nil,
					err:       err,
				}
			}

			return analyticsFetchedMsg{
				sessionID: sessionID,
				analytics: analytics,
				err:       nil,
			}
		}
	} else if inst.Tool == "gemini" {
		return func() tea.Msg {
			// Gemini analytics are updated via UpdateGeminiSession which is called in background
			// during UpdateStatus(). We just return the current snapshot.
			return analyticsFetchedMsg{
				sessionID:       sessionID,
				geminiAnalytics: inst.GeminiAnalytics,
				err:             nil,
			}
		}
	}

	return nil
}
