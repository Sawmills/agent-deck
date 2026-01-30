package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// WatchDialogMode represents the current mode of the watch dialog
type WatchDialogMode int

const (
	WatchModeList WatchDialogMode = iota
	WatchModeCreate
	WatchModeEdit
)

// WatchDialog manages watch goals (list, create, edit, pause, delete)
// TODO: Implement full CRUD functionality
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

	err error
}

// NewWatchDialog creates a new watch dialog
func NewWatchDialog(watchMgr *session.WatchManager) *WatchDialog {
	return &WatchDialog{
		watchMgr: watchMgr,
		mode:     WatchModeList,
	}
}

// Show makes the dialog visible
func (d *WatchDialog) Show() {
	d.visible = true
	// TODO: Load goals from watchMgr
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
	// TODO: Implement full update logic with:
	// - List mode: j/k navigation, Space to pause, d to delete, n to create, e to edit
	// - Create/Edit mode: form handling, Enter to submit
	// - Esc to close or return to list

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if d.mode == WatchModeList {
				d.visible = false
			} else {
				d.mode = WatchModeList
			}
			return d, nil
		}
	}

	return d, nil
}

// View implements tea.Model
func (d *WatchDialog) View() string {
	if !d.visible {
		return ""
	}

	// TODO: Implement full view with:
	// - List view: show all goals with status
	// - Create/Edit view: form with inputs
	// - Lipgloss styling

	switch d.mode {
	case WatchModeList:
		return "Watch Goals (TODO: Implement list view)\n\nPress 'n' to create, 'esc' to close"
	case WatchModeCreate:
		return "Create Watch Goal (TODO: Implement form)\n\nPress 'esc' to cancel"
	case WatchModeEdit:
		return "Edit Watch Goal (TODO: Implement form)\n\nPress 'esc' to cancel"
	}

	return ""
}

// SetSize updates the dialog dimensions
func (d *WatchDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}
