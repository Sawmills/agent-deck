package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	profile  string
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
// Profile is used for profile-isolated persistence.
func NewSessionObserver(profile string, config *AIObservationSettings) *SessionObserver {
	return &SessionObserver{
		sessions: make(map[string]*ObservedSession),
		config:   config,
		profile:  profile,
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

	o.mu.Unlock()

	// Persist observations to disk (outside lock to avoid blocking)
	if err := o.SaveObservations(o.profile, instance.ID); err != nil {
		log.Printf("Warning: failed to save observations for session %s: %v", instance.ID, err)
	}

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

// getObservationsPath returns the file path for observations of a session.
// Path: ~/.agent-deck/profiles/{profile}/observations/{sessionID}.json
// Profile-isolated to prevent cross-profile access.
func getObservationsPath(profile, sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is empty")
	}

	profileDir, err := GetProfileDir(profile)
	if err != nil {
		return "", err
	}

	obsDir := filepath.Join(profileDir, "observations")
	return filepath.Join(obsDir, sessionID+".json"), nil
}

// SaveObservations persists observations for a session to disk using atomic write pattern.
// Uses temp file + rename to prevent corruption on crash.
// Called after each observation is added to ensure durability.
func (o *SessionObserver) SaveObservations(profile, sessionID string) error {
	o.mu.RLock()
	observed, exists := o.sessions[sessionID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found in observer", sessionID)
	}

	// Get storage path
	path, err := getObservationsPath(profile, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get observations path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create observations directory: %w", err)
	}

	// Marshal observations to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(observed.Observations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal observations: %w", err)
	}

	// ═══════════════════════════════════════════════════════════════════
	// ATOMIC WRITE PATTERN: Prevents data corruption on crash/power loss
	// 1. Write to temporary file
	// 2. fsync the temp file (ensures data reaches disk)
	// 3. Atomic rename temp to final
	// ═══════════════════════════════════════════════════════════════════

	tmpPath := path + ".tmp"

	// Step 1: Write to temporary file (0600 = owner read/write only for security)
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Step 2: fsync the temp file to ensure data reaches disk before rename
	if err := syncFile(tmpPath); err != nil {
		// Log but don't fail - atomic rename still provides some safety
		log.Printf("Warning: fsync failed for %s: %v", tmpPath, err)
	}

	// Step 3: Atomic rename (this is atomic on POSIX systems)
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to finalize save: %w", err)
	}

	return nil
}

// LoadObservations loads observations for a session from disk.
// Automatically cleans up files older than 30 days.
// Returns nil if file doesn't exist (not an error - observations may not be persisted yet).
func (o *SessionObserver) LoadObservations(profile, sessionID string) error {
	// Get storage path
	path, err := getObservationsPath(profile, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get observations path: %w", err)
	}

	// Check if file exists
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		// File doesn't exist - not an error, just no observations yet
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat observations file: %w", err)
	}

	// Auto-cleanup: Delete files older than 30 days
	if time.Since(fileInfo.ModTime()) > 30*24*time.Hour {
		if err := os.Remove(path); err != nil {
			log.Printf("Warning: failed to delete old observations file %s: %v", path, err)
		}
		return nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read observations file: %w", err)
	}

	// Unmarshal JSON
	var observations []Observation
	if err := json.Unmarshal(data, &observations); err != nil {
		return fmt.Errorf("failed to unmarshal observations: %w", err)
	}

	// Load into observer
	o.mu.Lock()
	defer o.mu.Unlock()

	observed, exists := o.sessions[sessionID]
	if !exists {
		// Session not yet observed - create entry
		observed = &ObservedSession{
			Observations: observations,
		}
		o.sessions[sessionID] = observed
	} else {
		// Merge with existing observations (loaded ones first, then in-memory)
		// This handles the case where observations were loaded from disk
		// and new ones were added in memory since last save
		observed.Observations = observations
	}

	return nil
}
