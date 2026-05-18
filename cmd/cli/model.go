package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zenspanel/zenspanel/internal/config"
)

// screen identifies which sub-view is active. The root model holds the
// current screen and the per-screen sub-models. Switching between
// screens is just a state change — the bubbletea loop handles the
// re-render.
type screen int

const (
	screenMenu screen = iota
	screenStatus
	screenResetPassword
	screenCreateAdmin
	screenSuspend
	screenRestart
	screenLogs
	screenRebuild
)

type menuItem struct {
	label  string
	target screen
}

var menuItems = []menuItem{
	{"Status & Info", screenStatus},
	{"Reset Password", screenResetPassword},
	{"Create Admin", screenCreateAdmin},
	{"Suspend / Unsuspend User", screenSuspend},
	{"Restart Services", screenRestart},
	{"View Logs", screenLogs},
	{"Rebuild Frontend", screenRebuild},
}

type rootModel struct {
	cfg     *config.Config
	current screen
	cursor  int

	// sub-screen state — created lazily on first entry, reset when we
	// pop back to the menu
	statusModel    *statusModel
	resetModel     *resetPasswordModel
	createModel    *createAdminModel
	suspendModel   *suspendModel
	restartModel   *restartModel
	logsModel      *logsModel
	rebuildModel   *rebuildModel
	flashMessage   string
	flashIsError   bool
	width, height  int
}

func newRootModel(cfg *config.Config) *rootModel {
	return &rootModel{cfg: cfg, current: screenMenu}
}

func (m *rootModel) Init() tea.Cmd { return nil }

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		// Global quit. Sub-screens that need to swallow Ctrl+C/q can do
		// so before delegating; this branch only fires when the menu is
		// active or when the sub-screen returned without consuming the
		// key.
		if m.current == screenMenu {
			switch v.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(menuItems)-1 {
					m.cursor++
				}
			case "enter":
				m.flashMessage = ""
				return m, m.enterScreen(menuItems[m.cursor].target)
			}
			return m, nil
		}
	}

	// Delegate to the active sub-screen.
	switch m.current {
	case screenStatus:
		return m.updateStatus(msg)
	case screenResetPassword:
		return m.updateReset(msg)
	case screenCreateAdmin:
		return m.updateCreate(msg)
	case screenSuspend:
		return m.updateSuspend(msg)
	case screenRestart:
		return m.updateRestart(msg)
	case screenLogs:
		return m.updateLogs(msg)
	case screenRebuild:
		return m.updateRebuild(msg)
	}
	return m, nil
}

func (m *rootModel) View() string {
	switch m.current {
	case screenStatus:
		return m.viewStatus()
	case screenResetPassword:
		return m.viewReset()
	case screenCreateAdmin:
		return m.viewCreate()
	case screenSuspend:
		return m.viewSuspend()
	case screenRestart:
		return m.viewRestart()
	case screenLogs:
		return m.viewLogs()
	case screenRebuild:
		return m.viewRebuild()
	}
	return m.viewMenu()
}

func (m *rootModel) viewMenu() string {
	host, _ := hostname()
	header := fmt.Sprintf("ZensPanel CLI  v%s\nServer: %s", cliVersion, host)

	var body string
	for i, item := range menuItems {
		cursor := "  "
		label := item.label
		if i == m.cursor {
			cursor = selectedStyle.Render("▶ ")
			label = selectedStyle.Render(label)
		}
		body += cursor + label + "\n"
	}

	flash := ""
	if m.flashMessage != "" {
		st := successStyle
		if m.flashIsError {
			st = errorStyle
		}
		flash = "\n" + st.Render(m.flashMessage) + "\n"
	}

	help := helpStyle.Render("↑↓ navigate · enter select · q quit")

	content := titleStyle.Render(header) + "\n\n" + body + flash + help
	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m *rootModel) backToMenu(flash string, isErr bool) (tea.Model, tea.Cmd) {
	m.current = screenMenu
	m.flashMessage = flash
	m.flashIsError = isErr
	// drop sub-screen state so each entry is a fresh form
	m.statusModel = nil
	m.resetModel = nil
	m.createModel = nil
	m.suspendModel = nil
	m.restartModel = nil
	m.logsModel = nil
	m.rebuildModel = nil
	return m, nil
}

func (m *rootModel) enterScreen(s screen) tea.Cmd {
	m.current = s
	switch s {
	case screenStatus:
		m.statusModel = newStatusModel(m.cfg)
		return m.statusModel.Init()
	case screenResetPassword:
		m.resetModel = newResetPasswordModel()
	case screenCreateAdmin:
		m.createModel = newCreateAdminModel()
	case screenSuspend:
		m.suspendModel = newSuspendModel()
	case screenRestart:
		m.restartModel = newRestartModel()
	case screenLogs:
		m.logsModel = newLogsModel()
	case screenRebuild:
		m.rebuildModel = newRebuildModel()
	}
	return nil
}

func hostname() (string, error) {
	return osHostname()
}
