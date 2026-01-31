package ui

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// PreviewManager encapsulates preview cache state and async fetching logic.
// Preview content is fetched asynchronously to keep View() pure (no blocking I/O).
// The cache uses a 2-second TTL to balance live terminal updates with performance.
//
// Thread safety: previewCacheMu protects previewCache, previewCacheTime, and
// previewFetchingID. previewDebounceMu protects pendingPreviewID.
// Both mutexes move with this struct to maintain lock/data co-location.
//
// Embedded in Home for field access compatibility (same pattern as AnalyticsManager).
type PreviewManager struct {
	// Preview cache (async fetching - View() must be pure, no blocking I/O)
	previewCache      map[string]string    // sessionID -> cached preview content
	previewCacheTime  map[string]time.Time // sessionID -> when cached (for expiration)
	previewCacheMu    sync.RWMutex         // Protects previewCache for thread-safety
	previewFetchingID string               // ID currently being fetched (prevents duplicate fetches)

	// Preview debouncing (PERFORMANCE: prevents subprocess spawn on every keystroke)
	// During rapid navigation, we delay preview fetch by 150ms to let navigation settle
	pendingPreviewID  string     // Session ID waiting for debounced fetch
	previewDebounceMu sync.Mutex // Protects pendingPreviewID
}

// invalidatePreviewCache removes a session's preview from the cache.
// Called when session is deleted, renamed, or moved to ensure stale data is not displayed.
func (h *Home) invalidatePreviewCache(sessionID string) {
	h.previewCacheMu.Lock()
	delete(h.previewCache, sessionID)
	delete(h.previewCacheTime, sessionID)
	h.previewCacheMu.Unlock()
}

// fetchPreview returns a command that asynchronously fetches preview content.
// This keeps View() pure (no blocking I/O) as per Bubble Tea best practices.
func (h *Home) fetchPreview(inst *session.Instance) tea.Cmd {
	if inst == nil {
		return nil
	}
	sessionID := inst.ID
	return func() tea.Msg {
		content, err := inst.PreviewFull()
		return previewFetchedMsg{
			sessionID: sessionID,
			content:   content,
			err:       err,
		}
	}
}

// fetchPreviewDebounced returns a command that triggers preview fetch after debounce delay.
// PERFORMANCE: Prevents rapid subprocess spawning during keyboard navigation.
// The 150ms delay allows navigation to settle before spawning tmux capture-pane.
func (h *Home) fetchPreviewDebounced(sessionID string) tea.Cmd {
	const debounceDelay = 150 * time.Millisecond

	h.previewDebounceMu.Lock()
	h.pendingPreviewID = sessionID
	h.previewDebounceMu.Unlock()

	return func() tea.Msg {
		time.Sleep(debounceDelay)
		return previewDebounceMsg{sessionID: sessionID}
	}
}
