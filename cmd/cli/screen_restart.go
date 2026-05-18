package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type restartModel struct {
	services []string
	selected []bool
	cursor   int
	stage    int // 0 = pick, 1 = result
	results  []string
}

func newRestartModel() *restartModel {
	return &restartModel{
		services: []string{"zenspanel-api", "zenspanel-agent", "nginx", "mysql", "redis-server"},
		selected: []bool{true, true, false, false, false},
	}
}

func (m *rootModel) updateRestart(msg tea.Msg) (tea.Model, tea.Cmd) {
	r := m.restartModel
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "esc":
			return m.backToMenu("", false)
		case "up", "k":
			if r.stage == 0 && r.cursor > 0 {
				r.cursor--
			}
		case "down", "j":
			if r.stage == 0 && r.cursor < len(r.services)-1 {
				r.cursor++
			}
		case " ":
			if r.stage == 0 {
				r.selected[r.cursor] = !r.selected[r.cursor]
			}
		case "enter":
			if r.stage == 1 {
				return m.backToMenu("Restart complete", false)
			}
			r.runRestart()
			r.stage = 1
		}
	}
	return m, nil
}

func (r *restartModel) runRestart() {
	for i, svc := range r.services {
		if !r.selected[i] {
			continue
		}
		out, err := exec.Command("systemctl", "restart", svc).CombinedOutput()
		if err != nil {
			r.results = append(r.results, fmt.Sprintf("✗ %s: %s", svc, strings.TrimSpace(string(out))))
		} else {
			r.results = append(r.results, fmt.Sprintf("✓ %s restarted", svc))
		}
	}
}

func (m *rootModel) viewRestart() string {
	r := m.restartModel
	var b strings.Builder
	b.WriteString(headerStyle.Render("Restart Services") + "\n\n")

	if r.stage == 0 {
		b.WriteString(subtleStyle.Render("Select services to restart") + "\n\n")
		for i, svc := range r.services {
			cursor := "  "
			if i == r.cursor {
				cursor = selectedStyle.Render("▶ ")
			}
			box := "[ ]"
			if r.selected[i] {
				box = selectedStyle.Render("[✓]")
			}
			b.WriteString(cursor + box + " " + svc + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑↓ navigate · space toggle · enter execute · esc cancel"))
	} else {
		for _, line := range r.results {
			if strings.HasPrefix(line, "✓") {
				b.WriteString(successStyle.Render(line) + "\n")
			} else {
				b.WriteString(errorStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter back to menu"))
	}
	return boxStyle.Render(b.String())
}
