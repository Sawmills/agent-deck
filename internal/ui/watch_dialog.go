package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WatchDialogMode represents the current mode of the watch dialog
type WatchDialogMode int

const (
	WatchModeList WatchDialogMode = iota
	WatchModeCreate
	WatchModeEdit
)

// WatchDialog manages watch goals (list, create, edit, pause, delete)
type WatchDialog struct {
	visible  bool
	width    int
	height   int
	watchMgr *session.WatchManager
	goals    []*session.WatchGoal
	cursor   int
	mode     WatchDialogMode

	// Form fields
	nameInput     textinput.Model
	descInput     textinput.Model
	sessionsInput textinput.Model
	intervalInput textinput.Model
	focusIndex    int

	// For edit mode
	editingGoalID string

	err error
}

// NewWatchDialog creates a new watch dialog
func NewWatchDialog(watchMgr *session.WatchManager) *WatchDialog {
	// Initialize form inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "goal-name"
	nameInput.CharLimit = 50
	nameInput.Width = 40

	descInput := textinput.New()
	descInput.Placeholder = "What to watch for (e.g., 'alert on errors')"
	descInput.CharLimit = 200
	descInput.Width = 60

	sessionsInput := textinput.New()
	sessionsInput.Placeholder = "all or session-id-1,session-id-2"
	sessionsInput.CharLimit = 200
	sessionsInput.Width = 60
	sessionsInput.SetValue("all")

	intervalInput := textinput.New()
	intervalInput.Placeholder = "5"
	intervalInput.CharLimit = 10
	intervalInput.Width = 20
	intervalInput.SetValue("5")

	return &WatchDialog{
		watchMgr:      watchMgr,
		mode:          WatchModeList,
		nameInput:     nameInput,
		descInput:     descInput,
		sessionsInput: sessionsInput,
		intervalInput: intervalInput,
	}
}

// Show makes the dialog visible
func (d *WatchDialog) Show() {
	d.visible = true
	d.mode = WatchModeList
	d.cursor = 0
	d.err = nil

	// Load goals from watch manager
	if d.watchMgr != nil {
		d.goals = d.watchMgr.GetGoals()
	}
}

// Hide hides the dialog
func (d *WatchDialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *WatchDialog) IsVisible() bool {
	return d.visible
}

// Update implements tea.Model
func (d *WatchDialog) Update(msg tea.Msg) (*WatchDialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch d.mode {
		case WatchModeList:
			return d.handleListKeys(msg)
		case WatchModeCreate, WatchModeEdit:
			return d.handleFormKeys(msg)
		}
	}

	return d, nil
}

// handleListKeys handles keyboard input in list mode
func (d *WatchDialog) handleListKeys(msg tea.KeyMsg) (*WatchDialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.visible = false
		return d, nil

	case "j", "down":
		if d.cursor < len(d.goals)-1 {
			d.cursor++
		}

	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}

	case "n":
		// New goal - switch to create mode
		d.mode = WatchModeCreate
		d.focusIndex = 0
		d.clearForm()
		d.updateFormFocus()

	case "e":
		// Edit selected goal
		if len(d.goals) > 0 && d.cursor < len(d.goals) {
			goal := d.goals[d.cursor]
			d.mode = WatchModeEdit
			d.editingGoalID = goal.ID
			d.loadGoalIntoForm(goal)
			d.focusIndex = 0
			d.updateFormFocus()
		}

	case " ":
		// Toggle pause/resume
		if len(d.goals) > 0 && d.cursor < len(d.goals) {
			goal := d.goals[d.cursor]
			if goal.Paused {
				d.err = d.watchMgr.ResumeGoal(goal.ID)
			} else {
				d.err = d.watchMgr.PauseGoal(goal.ID)
			}
			// Reload goals
			d.goals = d.watchMgr.GetGoals()
		}

	case "d":
		// Delete selected goal
		if len(d.goals) > 0 && d.cursor < len(d.goals) {
			goal := d.goals[d.cursor]
			d.err = d.watchMgr.RemoveGoal(goal.ID)
			// Reload goals
			d.goals = d.watchMgr.GetGoals()
			// Adjust cursor if needed
			if d.cursor >= len(d.goals) && d.cursor > 0 {
				d.cursor--
			}
		}
	}

	return d, nil
}

// handleFormKeys handles keyboard input in create/edit mode
func (d *WatchDialog) handleFormKeys(msg tea.KeyMsg) (*WatchDialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to list mode
		d.mode = WatchModeList
		d.err = nil
		return d, nil

	case "tab":
		d.focusIndex++
		if d.focusIndex > 3 {
			d.focusIndex = 0
		}
		d.updateFormFocus()

	case "shift+tab":
		d.focusIndex--
		if d.focusIndex < 0 {
			d.focusIndex = 3
		}
		d.updateFormFocus()

	case "enter":
		// Submit form
		if d.mode == WatchModeCreate {
			d.err = d.createGoal()
		} else {
			d.err = d.updateGoal()
		}

		if d.err == nil {
			// Success - return to list mode
			d.mode = WatchModeList
			d.goals = d.watchMgr.GetGoals()
		}
		return d, nil

	default:
		// Update focused input
		var cmd tea.Cmd
		switch d.focusIndex {
		case 0:
			d.nameInput, cmd = d.nameInput.Update(msg)
		case 1:
			d.descInput, cmd = d.descInput.Update(msg)
		case 2:
			d.sessionsInput, cmd = d.sessionsInput.Update(msg)
		case 3:
			d.intervalInput, cmd = d.intervalInput.Update(msg)
		}
		return d, cmd
	}

	return d, nil
}

// updateFormFocus updates which input has focus
func (d *WatchDialog) updateFormFocus() {
	d.nameInput.Blur()
	d.descInput.Blur()
	d.sessionsInput.Blur()
	d.intervalInput.Blur()

	switch d.focusIndex {
	case 0:
		d.nameInput.Focus()
	case 1:
		d.descInput.Focus()
	case 2:
		d.sessionsInput.Focus()
	case 3:
		d.intervalInput.Focus()
	}
}

// clearForm clears all form inputs
func (d *WatchDialog) clearForm() {
	d.nameInput.SetValue("")
	d.descInput.SetValue("")
	d.sessionsInput.SetValue("all")
	d.intervalInput.SetValue("5")
	d.editingGoalID = ""
}

// loadGoalIntoForm loads a goal's data into the form
func (d *WatchDialog) loadGoalIntoForm(goal *session.WatchGoal) {
	d.nameInput.SetValue(goal.Name)
	d.descInput.SetValue(goal.Description)

	// Format sessions
	if len(goal.Sessions) == 0 {
		d.sessionsInput.SetValue("all")
	} else {
		d.sessionsInput.SetValue(strings.Join(goal.Sessions, ","))
	}

	// Format interval
	d.intervalInput.SetValue(fmt.Sprintf("%d", int(goal.Interval.Seconds())))
}

// createGoal creates a new watch goal from form data
func (d *WatchDialog) createGoal() error {
	name := strings.TrimSpace(d.nameInput.Value())
	if name == "" {
		return fmt.Errorf("name is required")
	}

	desc := strings.TrimSpace(d.descInput.Value())
	if desc == "" {
		return fmt.Errorf("description is required")
	}

	// Parse sessions
	var sessions []string
	sessionsStr := strings.TrimSpace(d.sessionsInput.Value())
	if sessionsStr != "all" && sessionsStr != "" {
		sessions = strings.Split(sessionsStr, ",")
		for i := range sessions {
			sessions[i] = strings.TrimSpace(sessions[i])
		}
	}

	// Parse interval
	intervalStr := strings.TrimSpace(d.intervalInput.Value())
	intervalSec, err := strconv.Atoi(intervalStr)
	if err != nil || intervalSec < 1 {
		return fmt.Errorf("interval must be a positive number")
	}

	goal := &session.WatchGoal{
		Name:        name,
		Description: desc,
		Sessions:    sessions,
		Interval:    time.Duration(intervalSec) * time.Second,
		Timeout:     1 * time.Hour, // Default timeout
		Action:      "notify",      // Default action
	}

	return d.watchMgr.AddGoal(goal)
}

// updateGoal updates an existing watch goal from form data
func (d *WatchDialog) updateGoal() error {
	name := strings.TrimSpace(d.nameInput.Value())
	if name == "" {
		return fmt.Errorf("name is required")
	}

	desc := strings.TrimSpace(d.descInput.Value())
	if desc == "" {
		return fmt.Errorf("description is required")
	}

	// Parse sessions
	var sessions []string
	sessionsStr := strings.TrimSpace(d.sessionsInput.Value())
	if sessionsStr != "all" && sessionsStr != "" {
		sessions = strings.Split(sessionsStr, ",")
		for i := range sessions {
			sessions[i] = strings.TrimSpace(sessions[i])
		}
	}

	// Parse interval
	intervalStr := strings.TrimSpace(d.intervalInput.Value())
	intervalSec, err := strconv.Atoi(intervalStr)
	if err != nil || intervalSec < 1 {
		return fmt.Errorf("interval must be a positive number")
	}

	goal := &session.WatchGoal{
		ID:          d.editingGoalID,
		Name:        name,
		Description: desc,
		Sessions:    sessions,
		Interval:    time.Duration(intervalSec) * time.Second,
		Timeout:     1 * time.Hour,
		Action:      "notify",
	}

	if err := d.watchMgr.RemoveGoal(d.editingGoalID); err != nil {
		return err
	}
	goal.ID = d.editingGoalID
	return d.watchMgr.AddGoal(goal)
}

// View implements tea.Model
func (d *WatchDialog) View() string {
	if !d.visible {
		return ""
	}

	switch d.mode {
	case WatchModeList:
		return d.viewList()
	case WatchModeCreate:
		return d.viewForm("Create Watch Goal")
	case WatchModeEdit:
		return d.viewForm("Edit Watch Goal")
	}

	return ""
}

// viewList renders the list view
func (d *WatchDialog) viewList() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)
	title := titleStyle.Render("Watch Goals")

	// Goals list
	var goalViews []string

	if len(d.goals) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Italic(true)
		goalViews = append(goalViews, emptyStyle.Render("No watch goals yet. Press 'n' to create one."))
	} else {
		for i, goal := range d.goals {
			// Status indicator
			var statusStyle lipgloss.Style
			var statusText string
			if goal.Paused {
				statusStyle = lipgloss.NewStyle().Foreground(ColorYellow)
				statusText = "⏸ PAUSED"
			} else {
				statusStyle = lipgloss.NewStyle().Foreground(ColorGreen)
				statusText = "● ACTIVE"
			}

			// Goal name
			nameStyle := lipgloss.NewStyle().Bold(true)
			if i == d.cursor {
				nameStyle = nameStyle.Foreground(ColorAccent)
			}

			// Build goal line
			cursor := "  "
			if i == d.cursor {
				cursor = "→ "
			}

			goalLine := fmt.Sprintf("%s%s %s", cursor, statusStyle.Render(statusText), nameStyle.Render(goal.Name))

			// Details
			detailStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
			details := fmt.Sprintf("    %s", goal.Description)
			if len(details) > 80 {
				details = details[:77] + "..."
			}

			// Stats
			sessionsText := "all sessions"
			if len(goal.Sessions) > 0 {
				sessionsText = fmt.Sprintf("%d sessions", len(goal.Sessions))
			}
			stats := fmt.Sprintf("    Interval: %ds | Scope: %s | Triggers: %d",
				int(goal.Interval.Seconds()), sessionsText, goal.TriggerCount)

			if !goal.LastTriggered.IsZero() {
				stats += fmt.Sprintf(" | Last: %s", goal.LastTriggered.Format("15:04:05"))
			}

			goalViews = append(goalViews, goalLine)
			goalViews = append(goalViews, detailStyle.Render(details))
			goalViews = append(goalViews, detailStyle.Render(stats))
			goalViews = append(goalViews, "") // Spacing
		}
	}

	goalsView := strings.Join(goalViews, "\n")

	// Error message
	var errorView string
	if d.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(ColorRed)
		errorView = errorStyle.Render(fmt.Sprintf("Error: %v", d.err))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)
	help := helpStyle.Render("n: new • e: edit • Space: pause/resume • d: delete • Esc: close")

	// Combine
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		goalsView,
		"",
		errorView,
		"",
		help,
	)

	// Border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	return borderStyle.Render(content)
}

// viewForm renders the create/edit form
func (d *WatchDialog) viewForm(formTitle string) string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)
	title := titleStyle.Render(formTitle)

	// Form labels
	labelStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	// Build form
	formLines := []string{
		labelStyle.Render("Name:"),
		d.nameInput.View(),
		"",
		labelStyle.Render("Description (what to watch for):"),
		d.descInput.View(),
		"",
		labelStyle.Render("Sessions (comma-separated IDs or 'all'):"),
		d.sessionsInput.View(),
		"",
		labelStyle.Render("Interval (seconds):"),
		d.intervalInput.View(),
	}

	formView := strings.Join(formLines, "\n")

	// Error message
	var errorView string
	if d.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(ColorRed)
		errorView = errorStyle.Render(fmt.Sprintf("Error: %v", d.err))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)
	help := helpStyle.Render("Tab: next field • Enter: save • Esc: cancel")

	// Combine
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		formView,
		"",
		errorView,
		"",
		help,
	)

	// Border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	return borderStyle.Render(content)
}

// SetSize updates the dialog dimensions
func (d *WatchDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}
