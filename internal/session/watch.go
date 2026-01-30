package session

import (
	"sync"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/ai"
)

type WatchAction string

const (
	WatchActionNotify  WatchAction = "notify"
	WatchActionSuggest WatchAction = "suggest"
)

type WatchGoal struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Sessions      []string      `json:"sessions"`
	Interval      time.Duration `json:"interval"`
	Timeout       time.Duration `json:"timeout"`
	Action        WatchAction   `json:"action"`
	Paused        bool          `json:"paused"`
	CreatedAt     time.Time     `json:"created_at"`
	LastTriggered time.Time     `json:"last_triggered"`
	TriggerCount  int           `json:"trigger_count"`
}

type WatchManager struct {
	goals      map[string]*WatchGoal
	mu         sync.RWMutex
	observer   *SessionObserver
	aiProvider ai.AIProvider
	config     *AIWatchSettings
	stopCh     chan struct{}
	wg         sync.WaitGroup
}
