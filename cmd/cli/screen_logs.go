package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type logsModel struct {
	files    []string
	cursor   int
	stage    int // 0 = pick, 1 = streaming
	viewport viewport.Model
	cmd      *exec.Cmd
	lines    []string
}

func newLogsModel() *logsModel {
	vp := viewport.New(80, 18)
	return &logsModel{
		files: []string{
			"/var/log/zenspanel/api.log",
			"/var/log/zenspanel/api-error.log",
			"/var/log/zenspanel/agent.log",
			"/var/log/zenspanel/agent-error.log",
			"/var/log/zenspanel/nginx-access.log",
			"/var/log/zenspanel/nginx-error.log",
		},
		viewport: vp,
	}
}

type logLineMsg string
type logEOFMsg struct{}

func (l *logsModel) startTail(path string) tea.Cmd {
	cmd := exec.Command("tail", "-n", "50", path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return func() tea.Msg { return logLineMsg("error: " + err.Error()) }
	}
	if err := cmd.Start(); err != nil {
		return func() tea.Msg { return logLineMsg("error: " + err.Error()) }
	}
	l.cmd = cmd
	return func() tea.Msg {
		// Read up to 200 lines or EOF, whichever comes first.
		scanner := bufio.NewScanner(stdout)
		var collected []string
		for i := 0; i < 200 && scanner.Scan(); i++ {
			collected = append(collected, scanner.Text())
		}
		_ = cmd.Wait()
		return logsLoadedMsg(collected)
	}
}

type logsLoadedMsg []string

func (m *rootModel) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	l := m.logsModel
	switch v := msg.(type) {
	case logsLoadedMsg:
		l.lines = []string(v)
		l.viewport.SetContent(strings.Join(l.lines, "\n"))
		l.viewport.GotoBottom()
		return m, nil
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q":
			if l.cmd != nil && l.cmd.Process != nil {
				_ = l.cmd.Process.Kill()
			}
			return m.backToMenu("", false)
		case "up", "k":
			if l.stage == 0 && l.cursor > 0 {
				l.cursor--
			}
		case "down", "j":
			if l.stage == 0 && l.cursor < len(l.files)-1 {
				l.cursor++
			}
		case "enter":
			if l.stage == 0 {
				l.stage = 1
				return m, l.startTail(l.files[l.cursor])
			}
		}
	}
	if l.stage == 1 {
		var cmd tea.Cmd
		l.viewport, cmd = l.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *rootModel) viewLogs() string {
	l := m.logsModel
	var b strings.Builder
	if l.stage == 0 {
		b.WriteString(headerStyle.Render("View Logs") + "\n\n")
		for i, f := range l.files {
			cursor := "  "
			label := f
			if i == l.cursor {
				cursor = selectedStyle.Render("▶ ")
				label = selectedStyle.Render(f)
			}
			b.WriteString(cursor + label + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑↓ navigate · enter open · esc back"))
	} else {
		b.WriteString(headerStyle.Render(fmt.Sprintf("Logs: %s", l.files[l.cursor])) + "\n\n")
		b.WriteString(l.viewport.View() + "\n\n")
		b.WriteString(helpStyle.Render("↑↓ scroll · esc/q back"))
	}
	return boxStyle.Render(b.String())
}
