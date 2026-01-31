package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/git"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// handleSearchKey handles keys when search is visible
func (h *Home) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		selected := h.search.Selected()
		if selected != nil {
			// Ensure the session's group AND all parent groups are expanded so it's visible
			if selected.GroupPath != "" {
				h.groupTree.ExpandGroupWithParents(selected.GroupPath)
			}
			h.rebuildFlatItems()

			// Find the session in flatItems (not instances) and set cursor
			for i, item := range h.flatItems {
				if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == selected.ID {
					h.cursor = i
					h.syncViewport() // Ensure the cursor is visible in the viewport
					break
				}
			}
		}
		h.search.Hide()
		return h, nil
	case "esc":
		h.search.Hide()
		return h, nil
	}

	var cmd tea.Cmd
	h.search, cmd = h.search.Update(msg)

	// Check if user wants to switch to global search
	if h.search.WantsSwitchToGlobal() && h.globalSearchIndex != nil {
		h.globalSearch.SetSize(h.width, h.height)
		h.globalSearch.Show()
	}

	return h, cmd
}

// handleGlobalSearchKey handles keys when global search is visible
func (h *Home) handleGlobalSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		selected := h.globalSearch.Selected()
		if selected != nil {
			h.globalSearch.Hide()
			return h, h.handleGlobalSearchSelection(selected)
		}
		h.globalSearch.Hide()
		return h, nil
	case "esc":
		h.globalSearch.Hide()
		return h, nil
	}

	var cmd tea.Cmd
	h.globalSearch, cmd = h.globalSearch.Update(msg)

	// Check if user wants to switch to local search
	if h.globalSearch.WantsSwitchToLocal() {
		h.search.SetItems(h.instances)
		h.search.Show()
	}

	return h, cmd
}

// handleGlobalSearchSelection handles selection from global search
func (h *Home) handleGlobalSearchSelection(result *GlobalSearchResult) tea.Cmd {
	// Check if session already exists in Agent Deck
	h.instancesMu.RLock()
	for _, inst := range h.instances {
		if inst.ClaudeSessionID == result.SessionID {
			h.instancesMu.RUnlock()
			// Jump to existing session
			h.jumpToSession(inst)
			return nil
		}
	}
	h.instancesMu.RUnlock()

	// Create new session with this Claude session ID
	return h.createSessionFromGlobalSearch(result)
}

// handleNewDialogKey handles keys when new dialog is visible
func (h *Home) handleNewDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate before creating session
		if validationErr := h.newDialog.Validate(); validationErr != "" {
			h.newDialog.SetError(validationErr)
			return h, nil
		}

		// Get values including worktree settings
		name, path, command, branchName, worktreeEnabled := h.newDialog.GetValuesWithWorktree()
		groupPath := h.newDialog.GetSelectedGroup()
		claudeOpts := h.newDialog.GetClaudeOptions() // Get Claude options if applicable

		// Handle worktree creation if enabled
		var worktreePath, worktreeRepoRoot string
		if worktreeEnabled && branchName != "" {
			// Validate path is a git repo
			if !git.IsGitRepo(path) {
				h.newDialog.SetError("Path is not a git repository")
				return h, nil
			}

			repoRoot, err := git.GetRepoRoot(path)
			if err != nil {
				h.newDialog.SetError(fmt.Sprintf("Failed to get repo root: %v", err))
				return h, nil
			}

			// Generate worktree path using configured location
			wtSettings := session.GetWorktreeSettings()
			worktreePath = git.GenerateWorktreePath(repoRoot, branchName, wtSettings.DefaultLocation)

			// Ensure parent directory exists (needed for subdirectory mode)
			if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
				h.newDialog.SetError(fmt.Sprintf("Failed to create parent directory: %v", err))
				return h, nil
			}

			// Create worktree
			if err := git.CreateWorktree(repoRoot, worktreePath, branchName); err != nil {
				h.newDialog.SetError(fmt.Sprintf("Failed to create worktree: %v", err))
				return h, nil
			}

			// Store repo root for later use
			worktreeRepoRoot = repoRoot
			// Update path to worktree for session creation
			path = worktreePath
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			h.newDialog.Hide()
			h.confirmDialog.ShowCreateDirectory(path, name, command, groupPath)
			return h, nil
		}

		h.newDialog.Hide()
		h.clearError()

		geminiYoloMode := h.newDialog.IsGeminiYoloMode()

		return h, h.createSessionInGroupWithWorktreeAndOptions(name, path, command, groupPath, worktreePath, worktreeRepoRoot, branchName, geminiYoloMode, claudeOpts)

	case "esc":
		h.newDialog.Hide()
		h.clearError() // Clear any validation error
		return h, nil
	}

	var cmd tea.Cmd
	h.newDialog, cmd = h.newDialog.Update(msg)
	return h, cmd
}

// handleMainKey handles keys in main view
func (h *Home) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return h.tryQuit()

	case "esc":
		// Dismiss maintenance banner if visible
		if h.maintenanceMsg != "" {
			h.maintenanceMsg = ""
			return h, nil
		}
		// Double ESC to quit (#28) - for non-English keyboard users
		// If ESC pressed twice within 500ms, quit the application
		if time.Since(h.lastEscTime) < 500*time.Millisecond {
			return h.tryQuit()
		}
		// First ESC - record time, show hint in status bar
		h.lastEscTime = time.Now()
		return h, nil

	case "up", "k":
		if h.cursor > 0 {
			h.cursor--
			h.syncViewport()
			// Track navigation for adaptive background updates
			h.lastNavigationTime = time.Now()
			h.isNavigating = true
			// PERFORMANCE: Debounced preview fetch - waits 150ms for navigation to settle
			// This prevents spawning tmux subprocess on every keystroke
			if selected := h.getSelectedSession(); selected != nil {
				return h, h.fetchPreviewDebounced(selected.ID)
			}
		}
		return h, nil

	case "down", "j":
		if h.cursor < len(h.flatItems)-1 {
			h.cursor++
			h.syncViewport()
			// Track navigation for adaptive background updates
			h.lastNavigationTime = time.Now()
			h.isNavigating = true
			// PERFORMANCE: Debounced preview fetch - waits 150ms for navigation to settle
			// This prevents spawning tmux subprocess on every keystroke
			if selected := h.getSelectedSession(); selected != nil {
				return h, h.fetchPreviewDebounced(selected.ID)
			}
		}
		return h, nil

	// Vi-style pagination (#38) - half/full page scrolling
	case "ctrl+u": // Half page up
		pageSize := h.getVisibleHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		h.cursor -= pageSize
		if h.cursor < 0 {
			h.cursor = 0
		}
		h.syncViewport()
		h.lastNavigationTime = time.Now()
		h.isNavigating = true
		if selected := h.getSelectedSession(); selected != nil {
			return h, h.fetchPreviewDebounced(selected.ID)
		}
		return h, nil

	case "ctrl+d": // Half page down
		pageSize := h.getVisibleHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		h.cursor += pageSize
		if h.cursor >= len(h.flatItems) {
			h.cursor = len(h.flatItems) - 1
		}
		if h.cursor < 0 {
			h.cursor = 0
		}
		h.syncViewport()
		h.lastNavigationTime = time.Now()
		h.isNavigating = true
		if selected := h.getSelectedSession(); selected != nil {
			return h, h.fetchPreviewDebounced(selected.ID)
		}
		return h, nil

	case "ctrl+b": // Full page up (backward)
		pageSize := h.getVisibleHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		h.cursor -= pageSize
		if h.cursor < 0 {
			h.cursor = 0
		}
		h.syncViewport()
		h.lastNavigationTime = time.Now()
		h.isNavigating = true
		if selected := h.getSelectedSession(); selected != nil {
			return h, h.fetchPreviewDebounced(selected.ID)
		}
		return h, nil

	case "ctrl+f": // Full page down (forward)
		pageSize := h.getVisibleHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		h.cursor += pageSize
		if h.cursor >= len(h.flatItems) {
			h.cursor = len(h.flatItems) - 1
		}
		if h.cursor < 0 {
			h.cursor = 0
		}
		h.syncViewport()
		h.lastNavigationTime = time.Now()
		h.isNavigating = true
		if selected := h.getSelectedSession(); selected != nil {
			return h, h.fetchPreviewDebounced(selected.ID)
		}
		return h, nil

	case "G": // Jump to bottom
		if len(h.flatItems) > 0 {
			h.cursor = len(h.flatItems) - 1
			h.syncViewport()
			h.lastNavigationTime = time.Now()
			h.isNavigating = true
			if selected := h.getSelectedSession(); selected != nil {
				return h, h.fetchPreviewDebounced(selected.ID)
			}
		}
		return h, nil

	case "enter":
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				// Block attachment during animations (must match renderPreviewPane display logic)
				if h.hasActiveAnimation(item.Session.ID) {
					h.setError(fmt.Errorf("session is starting, please wait..."))
					return h, nil
				}
				if item.Session.Exists() {
					h.isAttaching.Store(true) // Prevent View() output during transition (atomic)
					return h, h.attachSession(item.Session)
				}
			} else if item.Type == session.ItemTypeGroup {
				// Toggle group on enter
				groupPath := item.Path
				h.groupTree.ToggleGroup(groupPath)
				h.rebuildFlatItems()
				for i, fi := range h.flatItems {
					if fi.Type == session.ItemTypeGroup && fi.Path == groupPath {
						h.cursor = i
						break
					}
				}
			}
		}
		return h, nil

	case "tab", "l", "right":
		// Expand/collapse group or expand if on session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				groupPath := item.Path
				h.groupTree.ToggleGroup(groupPath)
				h.rebuildFlatItems()
				for i, fi := range h.flatItems {
					if fi.Type == session.ItemTypeGroup && fi.Path == groupPath {
						h.cursor = i
						break
					}
				}
			}
		}
		return h, nil

	case "h", "left":
		// Collapse group
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				groupPath := item.Path
				h.groupTree.CollapseGroup(groupPath)
				h.rebuildFlatItems()
				for i, fi := range h.flatItems {
					if fi.Type == session.ItemTypeGroup && fi.Path == groupPath {
						h.cursor = i
						break
					}
				}
			} else if item.Type == session.ItemTypeSession {
				// Move cursor to parent group
				h.groupTree.CollapseGroup(item.Path)
				h.rebuildFlatItems()
				// Find the group in flatItems
				for i, fi := range h.flatItems {
					if fi.Type == session.ItemTypeGroup && fi.Path == item.Path {
						h.cursor = i
						break
					}
				}
			}
		}
		return h, nil

	case "shift+up", "K":
		// Move item up
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.MoveGroupUp(item.Path)
			} else if item.Type == session.ItemTypeSession {
				h.groupTree.MoveSessionUp(item.Session)
			}
			h.rebuildFlatItems()
			if h.cursor > 0 {
				h.cursor--
			}
			h.saveInstances()
		}
		return h, nil

	case "shift+down", "J":
		// Move item down
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupTree.MoveGroupDown(item.Path)
			} else if item.Type == session.ItemTypeSession {
				h.groupTree.MoveSessionDown(item.Session)
			}
			h.rebuildFlatItems()
			if h.cursor < len(h.flatItems)-1 {
				h.cursor++
			}
			h.saveInstances()
		}
		return h, nil

	case "m":
		// Move session to different group
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession {
				h.groupDialog.ShowMove(h.groupTree.GetGroupNames())
			}
		}
		return h, nil

	case "f":
		// Quick fork session (same title with " (fork)" suffix)
		// Only available when session has a valid Claude session ID
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.CanFork() {
				return h, h.quickForkSession(item.Session)
			}
		}
		return h, nil

	case "F", "shift+f":
		// Fork with dialog (customize title and group)
		// Only available when session has a valid Claude session ID
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.CanFork() {
				return h, h.forkSessionWithDialog(item.Session)
			}
		}
		return h, nil

	case "M", "shift+m":
		// MCP Manager - for Claude and Gemini sessions
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil &&
				(item.Session.Tool == "claude" || item.Session.Tool == "gemini") {
				h.mcpDialog.SetSize(h.width, h.height)
				if err := h.mcpDialog.Show(item.Session.ProjectPath, item.Session.ID, item.Session.Tool); err != nil {
					h.setError(err)
				}
			}
		}
		return h, nil

	case "g":
		// Vi-style gg to jump to top (#38) - check for double-tap first
		if time.Since(h.lastGTime) < 500*time.Millisecond {
			// Double g - jump to top
			if len(h.flatItems) > 0 {
				h.cursor = 0
				h.syncViewport()
				h.lastNavigationTime = time.Now()
				h.isNavigating = true
				if selected := h.getSelectedSession(); selected != nil {
					return h, h.fetchPreviewDebounced(selected.ID)
				}
			}
			return h, nil
		}
		// Record time for potential gg detection
		h.lastGTime = time.Now()

		// Create new group with context-aware Tab toggle (Issue #111):
		// Defaults to subgroup when on a group/grouped session, root otherwise.
		// Tab toggle in the dialog lets users switch between Root and Subgroup.
		parentPath := ""
		parentName := ""
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				parentPath = item.Group.Path
				parentName = item.Group.Name
			} else if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.GroupPath != "" {
				parentPath = item.Session.GroupPath
				parentName = parentPath
				if idx := strings.LastIndex(parentPath, "/"); idx >= 0 {
					parentName = parentPath[idx+1:]
				}
			}
		}
		h.groupDialog.ShowCreateWithContext(parentPath, parentName)
		return h, nil

	case "r":
		// Rename group or session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				h.groupDialog.ShowRename(item.Path, item.Group.Name)
			} else if item.Type == session.ItemTypeSession && item.Session != nil {
				h.groupDialog.ShowRenameSession(item.Session.ID, item.Session.Title)
			}
		}
		return h, nil

	case "/":
		// Open global search first if available, otherwise local search
		if h.globalSearchIndex != nil {
			h.globalSearch.SetSize(h.width, h.height)
			h.globalSearch.Show()
		} else {
			h.search.Show()
		}
		return h, nil

	case "?":
		h.helpOverlay.SetSize(h.width, h.height)
		h.helpOverlay.Show()
		return h, nil

	case "S":
		// Open settings panel
		h.settingsPanel.Show()
		h.settingsPanel.SetSize(h.width, h.height)
		return h, nil

	case "n":
		// Collect unique project paths sorted by most recently accessed
		type pathInfo struct {
			path           string
			lastAccessedAt time.Time
		}
		pathMap := make(map[string]*pathInfo)
		for _, inst := range h.instances {
			if inst.ProjectPath == "" {
				continue
			}
			existing, ok := pathMap[inst.ProjectPath]
			if !ok {
				// First time seeing this path
				accessTime := inst.LastAccessedAt
				if accessTime.IsZero() {
					accessTime = inst.CreatedAt // Fall back to creation time
				}
				pathMap[inst.ProjectPath] = &pathInfo{
					path:           inst.ProjectPath,
					lastAccessedAt: accessTime,
				}
			} else {
				// Update if this instance was accessed more recently
				accessTime := inst.LastAccessedAt
				if accessTime.IsZero() {
					accessTime = inst.CreatedAt
				}
				if accessTime.After(existing.lastAccessedAt) {
					existing.lastAccessedAt = accessTime
				}
			}
		}

		// Convert to slice and sort by most recent first
		pathInfos := make([]*pathInfo, 0, len(pathMap))
		for _, info := range pathMap {
			pathInfos = append(pathInfos, info)
		}
		sort.Slice(pathInfos, func(i, j int) bool {
			return pathInfos[i].lastAccessedAt.After(pathInfos[j].lastAccessedAt)
		})

		// Extract sorted paths
		paths := make([]string, len(pathInfos))
		for i, info := range pathInfos {
			paths[i] = info.path
		}
		h.newDialog.SetPathSuggestions(paths)

		// Apply user's preferred default tool from config
		h.newDialog.SetDefaultTool(session.GetDefaultTool())

		// Auto-select parent group from current cursor position
		groupPath := session.DefaultGroupPath
		groupName := session.DefaultGroupName
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeGroup {
				groupPath = item.Group.Path
				groupName = item.Group.Name
			} else if item.Type == session.ItemTypeSession {
				// Use the session's group
				groupPath = item.Path
				if group, exists := h.groupTree.Groups[groupPath]; exists {
					groupName = group.Name
				}
			}
		}
		defaultPath := h.getDefaultPathForGroup(groupPath)
		h.newDialog.ShowInGroup(groupPath, groupName, defaultPath)
		return h, nil

	case "d":
		// Show confirmation dialog before deletion (prevents accidental deletion)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				h.confirmDialog.ShowDeleteSession(item.Session.ID, item.Session.Title)
			} else if item.Type == session.ItemTypeGroup && item.Path != session.DefaultGroupPath {
				h.confirmDialog.ShowDeleteGroup(item.Path, item.Group.Name)
			}
		}
		return h, nil

	case "i":
		return h, h.importSessions

	case "u":
		// Mark session as unread (change idle → waiting)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				tmuxSess := item.Session.GetTmuxSession()
				if tmuxSess != nil {
					tmuxSess.ResetAcknowledged()
					_ = item.Session.UpdateStatus()
					h.saveInstances()
				}
			}
		}
		return h, nil

	case "v":
		// Toggle preview mode (cycle: both → output-only → analytics-only → both)
		h.previewMode = (h.previewMode + 1) % 3
		return h, nil

	case "y":
		// Toggle Gemini YOLO mode (requires restart)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.Tool == "gemini" {
				inst := item.Session
				// Determine current YOLO state
				currentYolo := false
				if inst.GeminiYoloMode != nil {
					currentYolo = *inst.GeminiYoloMode
				} else {
					// Fall back to global config
					userConfig, _ := session.LoadUserConfig()
					if userConfig != nil {
						currentYolo = userConfig.Gemini.YoloMode
					}
				}
				// Toggle: set per-session override to opposite of current
				newYolo := !currentYolo
				inst.GeminiYoloMode = &newYolo
				h.saveInstances()
				// If session is running, it needs restart to apply
				if inst.Status == session.StatusRunning || inst.Status == session.StatusWaiting {
					h.resumingSessions[inst.ID] = time.Now()
					return h, h.restartSession(inst)
				}
			}
		}
		return h, nil

	case "R":
		// Restart session (Shift+R - recreate tmux session with resume)
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				if item.Session.CanRestart() {
					// Track as resuming for animation (before async call starts)
					h.resumingSessions[item.Session.ID] = time.Now()
					return h, h.restartSession(item.Session)
				}
			}
		}
		return h, nil

	case "c":
		// Copy last AI response to system clipboard
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				return h, h.copySessionOutput(item.Session)
			}
		}
		return h, nil

	case "x":
		// Send session output to another session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				others := h.getOtherActiveSessions(item.Session.ID)
				if len(others) == 0 {
					h.setError(fmt.Errorf("no other sessions to send to"))
					return h, nil
				}
				h.sessionPickerDialog.SetSize(h.width, h.height)
				h.sessionPickerDialog.Show(item.Session, h.instances)
			}
		}
		return h, nil

	case "ctrl+g":
		// Open Gemini model selection dialog (only for Gemini sessions)
		if inst := h.getSelectedSession(); inst != nil && inst.Tool == "gemini" {
			cmd := h.geminiModelDialog.Show(inst.ID, inst.GeminiModel)
			return h, cmd
		}
		return h, nil

	case "ctrl+z":
		// Undo last session delete (Chrome-style: restores in reverse order)
		if len(h.undoStack) == 0 {
			h.setError(fmt.Errorf("nothing to undo"))
			return h, nil
		}
		entry := h.undoStack[len(h.undoStack)-1]
		h.undoStack = h.undoStack[:len(h.undoStack)-1]
		inst := entry.instance
		return h, func() tea.Msg {
			err := inst.Restart()
			return sessionRestoredMsg{instance: inst, err: err}
		}

	case "ctrl+r":
		// Manual refresh (useful if watcher fails or for user preference)
		state := h.preserveState()

		cmd := func() tea.Msg {
			instances, groups, err := h.storage.LoadWithGroups()
			return loadSessionsMsg{
				instances:    instances,
				groups:       groups,
				err:          err,
				restoreState: &state,
			}
		}

		return h, cmd

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Quick jump to Nth root group (1-indexed)
		targetNum := int(msg.String()[0] - '0') // Convert "1" -> 1, "2" -> 2, etc.
		h.jumpToRootGroup(targetNum)
		return h, nil

	case "0":
		// Clear status filter (show all)
		h.statusFilter = ""
		h.rebuildFlatItems()
		return h, nil

	case "!", "shift+1":
		// Filter to running sessions only
		if h.statusFilter == session.StatusRunning {
			h.statusFilter = "" // Toggle off
		} else {
			h.statusFilter = session.StatusRunning
		}
		h.rebuildFlatItems()
		return h, nil

	case "@", "shift+2":
		// Filter to waiting sessions only
		if h.statusFilter == session.StatusWaiting {
			h.statusFilter = "" // Toggle off
		} else {
			h.statusFilter = session.StatusWaiting
		}
		h.rebuildFlatItems()
		return h, nil

	case "#", "shift+3":
		// Filter to idle sessions only
		if h.statusFilter == session.StatusIdle {
			h.statusFilter = "" // Toggle off
		} else {
			h.statusFilter = session.StatusIdle
		}
		h.rebuildFlatItems()
		return h, nil

	case "$", "shift+4":
		// Filter to error sessions only
		if h.statusFilter == session.StatusError {
			h.statusFilter = "" // Toggle off
		} else {
			h.statusFilter = session.StatusError
		}
		h.rebuildFlatItems()
		return h, nil

	case "A", "shift+a":
		if h.aiChatPanel != nil {
			if selected := h.getSelectedSession(); selected != nil {
				h.aiChatPanel.sessionID = selected.ID
				h.aiChatPanel.SetContentFetcher(func(sessionID string) (string, string, error) {
					h.instancesMu.RLock()
					inst := h.instanceByID[sessionID]
					h.instancesMu.RUnlock()
					if inst == nil {
						return "", "", fmt.Errorf("session not found")
					}
					content, _ := getSessionContent(inst)
					metadata := fmt.Sprintf("Tool: %s\nPath: %s\nStatus: %s", inst.Tool, inst.ProjectPath, inst.Status)
					return content, metadata, nil
				})
				h.aiChatPanel.SetSize(h.width, h.height)
				h.aiChatPanel.Show()
			}
		}
		return h, nil

	case "W", "shift+w":
		if h.watchDialog != nil {
			h.watchDialog.SetSize(h.width, h.height)
			h.watchDialog.Show()
		}
		return h, nil
	}

	return h, nil
}

// handleConfirmDialogKey handles keys when confirmation dialog is visible
func (h *Home) handleConfirmDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch h.confirmDialog.GetConfirmType() {
	case ConfirmQuitWithPool:
		switch msg.String() {
		case "k", "K":
			h.confirmDialog.Hide()
			h.isQuitting = true
			return h, h.performQuit(false)
		case "s", "S":
			h.confirmDialog.Hide()
			h.isQuitting = true
			return h, h.performQuit(true)
		case "esc":
			h.confirmDialog.Hide()
			h.isQuitting = false
			return h, nil
		}
		return h, nil

	case ConfirmCreateDirectory:
		switch msg.String() {
		case "y", "Y":
			name, path, command, groupPath := h.confirmDialog.GetPendingSession()
			h.confirmDialog.Hide()
			if err := os.MkdirAll(path, 0755); err != nil {
				h.setError(fmt.Errorf("failed to create directory: %w", err))
				return h, nil
			}
			return h, h.createSessionInGroupWithWorktreeAndOptions(name, path, command, groupPath, "", "", "", false, nil)
		case "n", "N", "esc":
			h.confirmDialog.Hide()
			return h, nil
		}
		return h, nil

	default:
		// Handle delete confirmations (session/group)
		switch msg.String() {
		case "y", "Y":
			// User confirmed - perform the deletion
			switch h.confirmDialog.GetConfirmType() {
			case ConfirmDeleteSession:
				sessionID := h.confirmDialog.GetTargetID()
				if inst := h.getInstanceByID(sessionID); inst != nil {
					h.confirmDialog.Hide()
					return h, h.deleteSession(inst)
				}
			case ConfirmDeleteGroup:
				groupPath := h.confirmDialog.GetTargetID()
				h.groupTree.DeleteGroup(groupPath)
				h.instancesMu.Lock()
				h.instances = h.groupTree.GetAllInstances()
				h.instancesMu.Unlock()
				h.rebuildFlatItems()
				h.saveInstances()
			}
			h.confirmDialog.Hide()
			return h, nil

		case "n", "N", "esc":
			// User cancelled
			h.confirmDialog.Hide()
			return h, nil
		}
	}

	return h, nil
}

// handleMCPDialogKey handles keys when MCP dialog is visible
func (h *Home) handleMCPDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// DEBUG: Log entry point
		log.Printf("[MCP-DEBUG] Enter pressed in MCP dialog")

		// Apply changes and close dialog
		hasChanged := h.mcpDialog.HasChanged()
		log.Printf("[MCP-DEBUG] HasChanged() = %v", hasChanged)

		if hasChanged {
			// Apply changes (saves state + writes .mcp.json)
			if err := h.mcpDialog.Apply(); err != nil {
				log.Printf("[MCP-DEBUG] Apply() failed: %v", err)
				h.setError(err)
				h.mcpDialog.Hide() // Hide dialog even on error
				return h, nil
			}
			log.Printf("[MCP-DEBUG] Apply() succeeded")

			// Find the session by ID (stored when dialog opened - same as Shift+S uses)
			sessionID := h.mcpDialog.GetSessionID()
			log.Printf("[MCP-DEBUG] Looking for sessionID: %q", sessionID)

			// O(1) lookup - no lock needed as Update() runs on main goroutine
			targetInst := h.getInstanceByID(sessionID)
			if targetInst != nil {
				log.Printf("[MCP-DEBUG] Found session by ID: %s, Title=%s", targetInst.ID, targetInst.Title)
			}

			if targetInst != nil {
				log.Printf("[MCP-DEBUG] Calling restartSession for: %s (with MCP loading animation)", targetInst.ID)
				// Track as MCP loading for animation in preview pane
				h.mcpLoadingSessions[targetInst.ID] = time.Now()
				// Set flag to skip MCP regeneration (Apply just wrote the config)
				targetInst.SkipMCPRegenerate = true
				// Restart the session to apply MCP changes
				h.mcpDialog.Hide()
				return h, h.restartSession(targetInst)
			} else {
				log.Printf("[MCP-DEBUG] No session found with ID: %s", sessionID)
			}
		}
		log.Printf("[MCP-DEBUG] Hiding dialog without restart")
		h.mcpDialog.Hide()
		return h, nil

	case "esc":
		h.mcpDialog.Hide()
		return h, nil

	default:
		h.mcpDialog.Update(msg)
		return h, nil
	}
}

// handleGroupDialogKey handles keys when group dialog is visible
func (h *Home) handleGroupDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate before proceeding
		if validationErr := h.groupDialog.Validate(); validationErr != "" {
			h.groupDialog.SetError(validationErr)
			return h, nil
		}
		h.clearError() // Clear any previous validation error

		switch h.groupDialog.Mode() {
		case GroupDialogCreate:
			name := h.groupDialog.GetValue()
			if name != "" {
				if h.groupDialog.HasParent() {
					// Create subgroup under parent
					parentPath := h.groupDialog.GetParentPath()
					h.groupTree.CreateSubgroup(parentPath, name)
				} else {
					// Create root-level group
					h.groupTree.CreateGroup(name)
				}
				h.rebuildFlatItems()
				h.saveInstances() // Persist the new group
			}
		case GroupDialogRename:
			name := h.groupDialog.GetValue()
			if name != "" {
				h.groupTree.RenameGroup(h.groupDialog.GetGroupPath(), name)
				h.instancesMu.Lock()
				h.instances = h.groupTree.GetAllInstances()
				h.instancesMu.Unlock()
				h.rebuildFlatItems()
				h.saveInstances()
			}
		case GroupDialogMove:
			groupName := h.groupDialog.GetSelectedGroup()
			if groupName != "" && h.cursor < len(h.flatItems) {
				item := h.flatItems[h.cursor]
				if item.Type == session.ItemTypeSession {
					// Find the group path from name
					for _, g := range h.groupTree.GroupList {
						if g.Name == groupName {
							h.groupTree.MoveSessionToGroup(item.Session, g.Path)
							h.instancesMu.Lock()
							h.instances = h.groupTree.GetAllInstances()
							h.instancesMu.Unlock()
							h.rebuildFlatItems()
							h.saveInstances()
							break
						}
					}
				}
			}
		case GroupDialogRenameSession:
			newName := h.groupDialog.GetValue()
			if newName != "" {
				sessionID := h.groupDialog.GetSessionID()
				// Find and rename the session (O(1) lookup)
				if inst := h.getInstanceByID(sessionID); inst != nil {
					inst.Title = newName
					inst.SyncTmuxDisplayName()
				}
				// Invalidate preview cache since title changed
				h.invalidatePreviewCache(sessionID)
				h.rebuildFlatItems()
				h.saveInstances()
			}
		}
		h.groupDialog.Hide()
		return h, nil
	case "esc":
		h.groupDialog.Hide()
		h.clearError() // Clear any validation error
		return h, nil
	}

	var cmd tea.Cmd
	h.groupDialog, cmd = h.groupDialog.Update(msg)
	return h, cmd
}

// handleForkDialogKey handles keyboard input for the fork dialog
func (h *Home) handleForkDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate before proceeding
		if validationErr := h.forkDialog.Validate(); validationErr != "" {
			h.forkDialog.SetError(validationErr)
			return h, nil
		}

		// Get fork parameters from dialog
		title, groupPath := h.forkDialog.GetValues()
		opts := h.forkDialog.GetOptions()
		h.clearError() // Clear any previous error

		// Find the currently selected session
		if h.cursor < len(h.flatItems) {
			item := h.flatItems[h.cursor]
			if item.Type == session.ItemTypeSession && item.Session != nil {
				h.forkDialog.Hide()
				return h, h.forkSessionCmdWithOptions(item.Session, title, groupPath, opts)
			}
		}
		h.forkDialog.Hide()
		return h, nil

	case "esc":
		h.forkDialog.Hide()
		h.clearError() // Clear any error
		return h, nil
	}

	var cmd tea.Cmd
	h.forkDialog, cmd = h.forkDialog.Update(msg)
	return h, cmd
}

// handleSessionPickerDialogKey handles key events when the session picker is visible.
func (h *Home) handleSessionPickerDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		selected := h.sessionPickerDialog.GetSelected()
		source := h.sessionPickerDialog.GetSource()
		h.sessionPickerDialog.Hide()
		if selected != nil && source != nil {
			return h, h.sendOutputToSession(source, selected)
		}
		return h, nil
	case "esc":
		h.sessionPickerDialog.Hide()
		return h, nil
	default:
		h.sessionPickerDialog.Update(msg)
		return h, nil
	}
}
