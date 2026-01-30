package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Default observation limits
const (
	// DefaultMaxObservationSize is the maximum size in bytes for a single observation (50KB)
	DefaultMaxObservationSize = 51200

	// DefaultMaxObservationsPerSession is the maximum observations to retain per session (FIFO)
	DefaultMaxObservationsPerSession = 100
)

// SessionObserver tracks terminal content observations across sessions.
// It captures pane content periodically and maintains a ring buffer of observations
// per session for later analysis and AI context retrieval.
type SessionObserver struct {
	sessions map[string]*ObservedSession
	mu       sync.RWMutex
	config   *AIObservationSettings
}

// ObservedSession tracks observations for a single session.
type ObservedSession struct {
	Instance     *Instance
	Observations []Observation // Ring buffer (FIFO eviction at retention limit)
	LastObserved time.Time
	ContentHash  string // Hash of last observed content (for change detection)
}

// Observation represents a single captured snapshot of terminal content.
type Observation struct {
	Timestamp   time.Time `json:"timestamp"`
	Content     string    `json:"content"`      // Truncated to MaxSizeBytes
	ContentHash string    `json:"content_hash"` // SHA256 of full content (before truncation)
	Status      Status    `json:"status"`       // Session status at observation time
}

// NewSessionObserver creates a new SessionObserver with the given configuration.
// If config is nil, defaults are used.
func NewSessionObserver(config *AIObservationSettings) *SessionObserver {
	return &SessionObserver{
		sessions: make(map[string]*ObservedSession),
		config:   config,
	}
}

// maxSizeBytes returns the configured max observation size or default.
func (o *SessionObserver) maxSizeBytes() int {
	if o.config != nil && o.config.MaxSizeBytes != nil {
		return *o.config.MaxSizeBytes
	}
	return DefaultMaxObservationSize
}

// retentionCount returns the configured retention count or default.
func (o *SessionObserver) retentionCount() int {
	if o.config != nil && o.config.RetentionCount != nil {
		return *o.config.RetentionCount
	}
	return DefaultMaxObservationsPerSession
}

// Observe captures the current terminal content for a session.
// It uses content hashing to detect changes and only stores observations
// when content differs from the previous observation.
//
// Integration hook: Call this from Instance.UpdateStatus() (instance.go:1177)
// after status detection completes:
//
//	func (i *Instance) UpdateStatus() error {
//	    // ... existing status detection ...
//
//	    // Observation hook (when observer is wired up):
//	    // if observer != nil {
//	    //     observer.Observe(i.ID)
//	    // }
//	    return nil
//	}
func (o *SessionObserver) Observe(instance *Instance) error {
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}

	// Capture pane content from tmux session
	tmuxSession := instance.GetTmuxSession()
	if tmuxSession == nil {
		return fmt.Errorf("session %s has no tmux session", instance.ID)
	}

	content, err := tmuxSession.CapturePane()
	if err != nil {
		return fmt.Errorf("failed to capture pane for session %s: %w", instance.ID, err)
	}

	// Calculate SHA256 hash of full content (before truncation)
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	o.mu.Lock()
	defer o.mu.Unlock()

	// Get or create observed session
	observed, exists := o.sessions[instance.ID]
	if !exists {
		observed = &ObservedSession{
			Instance:     instance,
			Observations: make([]Observation, 0, o.retentionCount()),
		}
		o.sessions[instance.ID] = observed
	}

	// Update instance reference (may have changed)
	observed.Instance = instance
	observed.LastObserved = time.Now()

	// Skip if content hasn't changed
	if hashStr == observed.ContentHash {
		return nil
	}

	// Content changed - create new observation
	observed.ContentHash = hashStr

	// Truncate content if exceeds max size
	truncatedContent := content
	maxSize := o.maxSizeBytes()
	if len(content) > maxSize {
		truncatedContent = content[:maxSize]
	}

	obs := Observation{
		Timestamp:   time.Now(),
		Content:     truncatedContent,
		ContentHash: hashStr,
		Status:      instance.Status,
	}

	// Add to ring buffer with FIFO eviction
	retentionLimit := o.retentionCount()
	if len(observed.Observations) >= retentionLimit {
		// Evict oldest (first element) - shift left
		copy(observed.Observations, observed.Observations[1:])
		observed.Observations = observed.Observations[:len(observed.Observations)-1]
	}
	observed.Observations = append(observed.Observations, obs)

	return nil
}

// GetObservations returns all observations for a session.
// Returns nil if session has not been observed.
func (o *SessionObserver) GetObservations(sessionID string) []Observation {
	o.mu.RLock()
	defer o.mu.RUnlock()

	observed, exists := o.sessions[sessionID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]Observation, len(observed.Observations))
	copy(result, observed.Observations)
	return result
}

// GetLatestObservation returns the most recent observation for a session.
// Returns nil if session has no observations.
func (o *SessionObserver) GetLatestObservation(sessionID string) *Observation {
	o.mu.RLock()
	defer o.mu.RUnlock()

	observed, exists := o.sessions[sessionID]
	if !exists || len(observed.Observations) == 0 {
		return nil
	}

	// Return copy of latest
	latest := observed.Observations[len(observed.Observations)-1]
	return &latest
}

// RemoveSession removes all observations for a session.
// Called when a session is deleted.
func (o *SessionObserver) RemoveSession(sessionID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.sessions, sessionID)
}

// ObservedSessionCount returns the number of sessions being observed.
func (o *SessionObserver) ObservedSessionCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.sessions)
}

// GetObservedSession returns the ObservedSession for a session ID.
// Returns nil if not found.
func (o *SessionObserver) GetObservedSession(sessionID string) *ObservedSession {
	o.mu.RLock()
	defer o.mu.RUnlock()

	observed, exists := o.sessions[sessionID]
	if !exists {
		return nil
	}
	return observed
}
