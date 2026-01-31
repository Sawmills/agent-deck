package session

// SessionProvider defines the interface for session operations
// used by the UI layer. This enables testing without real sessions.
//
// SessionProvider abstracts the core session operations needed by UI components,
// allowing them to work with mock implementations during testing without requiring
// actual tmux sessions or file system interactions.
type SessionProvider interface {
	// GetID returns the unique identifier for this session
	GetID() string

	// GetName returns the display name (Title) of this session
	GetName() string

	// GetStatus returns the current status of the session
	GetStatus() Status

	// GetPath returns the project path where this session operates
	GetPath() string

	// GetToolType returns the tool type (e.g., "claude", "gemini", "opencode")
	GetToolType() string

	// IsRunning returns true if the session is currently running
	IsRunning() bool

	// Start initiates the session
	Start() error

	// Stop terminates the session
	Stop() error
}

// Verify that Instance implements SessionProvider at compile time.
// This ensures that any changes to the interface are caught immediately.
var _ SessionProvider = (*Instance)(nil)
