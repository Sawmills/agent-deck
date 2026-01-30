package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

const (
	defaultWatchIntervalSeconds = 5
	defaultWatchTimeoutSeconds  = 3600
	maxConcurrentGoals          = 10
)

// NewWatchManager creates a new WatchManager.
func NewWatchManager(observer *SessionObserver, aiProvider ai.AIProvider, config *AIWatchSettings) *WatchManager {
	return &WatchManager{
		goals:      make(map[string]*WatchGoal),
		observer:   observer,
		aiProvider: aiProvider,
		config:     config,
	}
}

// AddGoal validates and adds a goal to the manager.
func (w *WatchManager) AddGoal(goal *WatchGoal) error {
	if goal == nil {
		return fmt.Errorf("goal is nil")
	}

	newGoal := *goal
	newGoal.Description = strings.TrimSpace(newGoal.Description)
	newGoal.Sessions = append([]string(nil), goal.Sessions...)

	if newGoal.Description == "" {
		return fmt.Errorf("goal description is empty")
	}
	if len(newGoal.Sessions) == 0 {
		return fmt.Errorf("goal sessions are empty")
	}
	if newGoal.ID == "" {
		newGoal.ID = generateID()
	}
	if newGoal.Interval <= 0 {
		newGoal.Interval = w.defaultInterval()
	}
	if newGoal.Timeout <= 0 {
		newGoal.Timeout = w.defaultTimeout()
	}
	if newGoal.CreatedAt.IsZero() {
		newGoal.CreatedAt = time.Now()
	}
	if newGoal.Action != WatchActionNotify && newGoal.Action != WatchActionSuggest {
		newGoal.Action = WatchActionNotify
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.goals[newGoal.ID]; exists {
		return fmt.Errorf("goal %s already exists", newGoal.ID)
	}
	if len(w.goals) >= w.maxConcurrentGoals() {
		return fmt.Errorf("max concurrent goals reached (%d)", w.maxConcurrentGoals())
	}

	w.goals[newGoal.ID] = &newGoal
	return nil
}

// RemoveGoal removes a goal by ID.
func (w *WatchManager) RemoveGoal(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("goal id is empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.goals[id]; !exists {
		return fmt.Errorf("goal %s not found", id)
	}
	delete(w.goals, id)
	return nil
}

// PauseGoal pauses a goal by ID.
func (w *WatchManager) PauseGoal(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("goal id is empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	goal, exists := w.goals[id]
	if !exists {
		return fmt.Errorf("goal %s not found", id)
	}
	goal.Paused = true
	return nil
}

// ResumeGoal resumes a paused goal by ID.
func (w *WatchManager) ResumeGoal(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("goal id is empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	goal, exists := w.goals[id]
	if !exists {
		return fmt.Errorf("goal %s not found", id)
	}
	goal.Paused = false
	return nil
}

// GetGoals returns a snapshot of all goals.
func (w *WatchManager) GetGoals() []*WatchGoal {
	w.mu.RLock()
	defer w.mu.RUnlock()

	goals := make([]*WatchGoal, 0, len(w.goals))
	for _, goal := range w.goals {
		clone := *goal
		clone.Sessions = append([]string(nil), goal.Sessions...)
		goals = append(goals, &clone)
	}
	return goals
}

// Start launches workers for all goals.
func (w *WatchManager) Start() {
	w.mu.Lock()
	if w.stopCh != nil {
		w.mu.Unlock()
		return
	}
	w.stopCh = make(chan struct{})
	goals := make([]*WatchGoal, 0, len(w.goals))
	for _, goal := range w.goals {
		goals = append(goals, goal)
	}
	w.mu.Unlock()

	for _, goal := range goals {
		w.wg.Add(1)
		go w.runGoalWorker(goal)
	}
}

// Stop gracefully shuts down all workers.
func (w *WatchManager) Stop() {
	w.mu.Lock()
	if w.stopCh == nil {
		w.mu.Unlock()
		return
	}
	close(w.stopCh)
	w.mu.Unlock()

	w.wg.Wait()

	w.mu.Lock()
	w.stopCh = nil
	w.mu.Unlock()
}

// evaluateGoal checks observations and triggers actions when needed.
func (w *WatchManager) evaluateGoal(goal *WatchGoal) error {
	if goal == nil {
		return fmt.Errorf("goal is nil")
	}
	if w.observer == nil {
		return fmt.Errorf("observer is nil")
	}
	if w.aiProvider == nil {
		return fmt.Errorf("ai provider is nil")
	}

	w.mu.RLock()
	goalID := goal.ID
	description := strings.TrimSpace(goal.Description)
	sessions := append([]string(nil), goal.Sessions...)
	w.mu.RUnlock()

	if description == "" {
		return fmt.Errorf("goal description is empty")
	}
	if len(sessions) == 0 {
		return fmt.Errorf("goal sessions are empty")
	}

	var observationsBuilder strings.Builder
	for _, sessionID := range sessions {
		if strings.TrimSpace(sessionID) == "" {
			continue
		}
		latest := w.observer.GetLatestObservation(sessionID)
		if latest == nil {
			continue
		}
		content := strings.TrimSpace(latest.Content)
		if content == "" {
			continue
		}
		if observationsBuilder.Len() > 0 {
			observationsBuilder.WriteString("\n\n")
		}
		observationsBuilder.WriteString("Session ")
		observationsBuilder.WriteString(sessionID)
		observationsBuilder.WriteString(" (")
		observationsBuilder.WriteString(latest.Timestamp.Format(time.RFC3339))
		observationsBuilder.WriteString("):\n")
		observationsBuilder.WriteString(content)
	}

	if observationsBuilder.Len() == 0 {
		return nil
	}

	prompt := fmt.Sprintf(
		"Goal: %s\nSession content: %s\nShould I take action? Reply <NoComment> if no, otherwise explain.",
		description,
		observationsBuilder.String(),
	)

	response, err := w.aiProvider.Chat(context.Background(), []ai.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return fmt.Errorf("ai chat failed: %w", err)
	}

	if strings.Contains(response, "<NoComment>") {
		return nil
	}

	response = strings.TrimSpace(response)

	w.mu.Lock()
	currentGoal, exists := w.goals[goalID]
	if exists {
		currentGoal.LastTriggered = time.Now()
		currentGoal.TriggerCount++
	}
	action := WatchActionNotify
	if exists {
		action = currentGoal.Action
	}
	w.mu.Unlock()

	switch action {
	case WatchActionSuggest:
		log.Printf("Watch goal %s triggered (suggest): %s", goalID, response)
	default:
		log.Printf("Watch goal %s triggered (notify): %s", goalID, response)
	}

	return nil
}

// runGoalWorker runs a worker loop for a single goal.
func (w *WatchManager) runGoalWorker(goal *WatchGoal) {
	defer w.wg.Done()

	w.mu.RLock()
	stopCh := w.stopCh
	w.mu.RUnlock()
	if stopCh == nil {
		return
	}

	interval := goal.Interval
	if interval <= 0 {
		interval = w.defaultInterval()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			w.mu.RLock()
			currentGoal, exists := w.goals[goal.ID]
			if !exists {
				w.mu.RUnlock()
				return
			}
			paused := currentGoal.Paused
			timeout := currentGoal.Timeout
			createdAt := currentGoal.CreatedAt
			currentID := currentGoal.ID
			w.mu.RUnlock()
			if paused {
				continue
			}
			if timeout > 0 && !createdAt.IsZero() && time.Since(createdAt) > timeout {
				if err := w.PauseGoal(currentID); err != nil {
					log.Printf("Watch goal %s pause error: %v", currentID, err)
				}
				continue
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("WATCH WORKER PANIC (recovered): %v", r)
					}
				}()
				if err := w.evaluateGoal(currentGoal); err != nil {
					log.Printf("Watch goal %s evaluation error: %v", currentGoal.ID, err)
				}
			}()
		}
	}
}

// SaveGoals persists watch goals to disk using atomic write pattern.
func (w *WatchManager) SaveGoals() error {
	profile := GetEffectiveProfile("")
	path, err := getWatchGoalsPath(profile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create watch goals directory: %w", err)
	}

	w.mu.RLock()
	goals := make([]*WatchGoal, 0, len(w.goals))
	for _, goal := range w.goals {
		clone := *goal
		clone.Sessions = append([]string(nil), goal.Sessions...)
		goals = append(goals, &clone)
	}
	w.mu.RUnlock()

	jsonData, err := json.MarshalIndent(goals, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal watch goals: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := syncFile(tmpPath); err != nil {
		log.Printf("Warning: fsync failed for %s: %v", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to finalize save: %w", err)
	}

	return nil
}

// LoadGoals loads watch goals from disk.
func (w *WatchManager) LoadGoals() error {
	profile := GetEffectiveProfile("")
	path, err := getWatchGoalsPath(profile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to stat watch goals file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read watch goals file: %w", err)
	}

	var goals []*WatchGoal
	if err := json.Unmarshal(data, &goals); err != nil {
		return fmt.Errorf("failed to unmarshal watch goals: %w", err)
	}

	w.mu.Lock()
	w.goals = make(map[string]*WatchGoal)
	w.mu.Unlock()

	for _, goal := range goals {
		if goal == nil {
			continue
		}
		if err := w.AddGoal(goal); err != nil {
			log.Printf("Warning: failed to load watch goal %s: %v", goal.ID, err)
		}
	}

	return nil
}

func (w *WatchManager) defaultInterval() time.Duration {
	if w.config != nil && w.config.DefaultInterval != nil && *w.config.DefaultInterval > 0 {
		return time.Duration(*w.config.DefaultInterval) * time.Second
	}
	return time.Duration(defaultWatchIntervalSeconds) * time.Second
}

func (w *WatchManager) defaultTimeout() time.Duration {
	if w.config != nil && w.config.DefaultTimeout != nil && *w.config.DefaultTimeout > 0 {
		return time.Duration(*w.config.DefaultTimeout) * time.Second
	}
	return time.Duration(defaultWatchTimeoutSeconds) * time.Second
}

func (w *WatchManager) maxConcurrentGoals() int {
	limit := maxConcurrentGoals
	if w.config != nil && w.config.MaxConcurrentGoals != nil && *w.config.MaxConcurrentGoals > 0 {
		limit = *w.config.MaxConcurrentGoals
	}
	if limit > maxConcurrentGoals {
		return maxConcurrentGoals
	}
	return limit
}

func getWatchGoalsPath(profile string) (string, error) {
	profileDir, err := GetProfileDir(profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(profileDir, "watch_goals.json"), nil
}
