package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type rebuildModel struct {
	choice  int // 0 = both, 1 = admin, 2 = user
	options []string
	stage   int // 0 = pick, 1 = running, 2 = done
	output  []string
	isError bool
}

func newRebuildModel() *rebuildModel {
	return &rebuildModel{
		options: []string{"Admin + User", "Admin only", "User only"},
	}
}

type rebuildDoneMsg struct {
	output  string
	isError bool
}

func (r *rebuildModel) run() tea.Cmd {
	return func() tea.Msg {
		var filter []string
		switch r.choice {
		case 1:
			filter = []string{"--filter", "@zenspanel/admin"}
		case 2:
			filter = []string{"--filter", "@zenspanel/user"}
		default:
			filter = []string{"-r"}
		}
		args := append([]string{}, filter...)
		args = append(args, "build")
		cmd := exec.Command("pnpm", args...)
		cmd.Dir = "/opt/zenspanel/src/frontend"
		out, err := cmd.CombinedOutput()
		if err != nil {
			return rebuildDoneMsg{output: string(out) + "\nERROR: " + err.Error(), isError: true}
		}
		// Copy dist to deployment locations.
		copyMap := map[string]string{
			"admin": "/opt/zenspanel/src/frontend/apps/admin/dist",
			"user":  "/opt/zenspanel/src/frontend/apps/user/dist",
		}
		for app, src := range copyMap {
			if r.choice == 1 && app != "admin" {
				continue
			}
			if r.choice == 2 && app != "user" {
				continue
			}
			dst := filepath.Join("/opt/zenspanel/frontend", app)
			if err := os.RemoveAll(dst); err != nil {
				return rebuildDoneMsg{output: string(out) + "\nremove old dist: " + err.Error(), isError: true}
			}
			cp := exec.Command("cp", "-r", src, dst)
			if cpOut, err := cp.CombinedOutput(); err != nil {
				return rebuildDoneMsg{output: string(out) + "\ncopy: " + err.Error() + "\n" + string(cpOut), isError: true}
			}
		}
		// Reload nginx so any new chunk hashes are picked up cleanly.
		_ = exec.Command("systemctl", "reload", "nginx").Run()
		return rebuildDoneMsg{output: string(out), isError: false}
	}
}

func (m *rootModel) updateRebuild(msg tea.Msg) (tea.Model, tea.Cmd) {
	r := m.rebuildModel
	switch v := msg.(type) {
	case rebuildDoneMsg:
		r.stage = 2
		r.isError = v.isError
		r.output = strings.Split(v.output, "\n")
		// Cap to last 30 lines so the box doesn't blow up.
		if len(r.output) > 30 {
			r.output = r.output[len(r.output)-30:]
		}
		return m, nil
	case tea.KeyMsg:
		switch v.String() {
		case "esc":
			if r.stage == 1 {
				return m, nil
			}
			return m.backToMenu("", false)
		case "up", "k":
			if r.stage == 0 && r.choice > 0 {
				r.choice--
			}
		case "down", "j":
			if r.stage == 0 && r.choice < len(r.options)-1 {
				r.choice++
			}
		case "enter":
			if r.stage == 0 {
				r.stage = 1
				return m, r.run()
			}
			if r.stage == 2 {
				flash := "Frontend rebuilt"
				if r.isError {
					flash = "Frontend rebuild failed"
				}
				return m.backToMenu(flash, r.isError)
			}
		}
	}
	return m, nil
}

func (m *rootModel) viewRebuild() string {
	r := m.rebuildModel
	var b strings.Builder
	b.WriteString(headerStyle.Render("Rebuild Frontend") + "\n\n")

	switch r.stage {
	case 0:
		b.WriteString(subtleStyle.Render("Select what to rebuild") + "\n\n")
		for i, o := range r.options {
			cursor := "  "
			label := o
			if i == r.choice {
				cursor = selectedStyle.Render("▶ ")
				label = selectedStyle.Render(o)
			}
			b.WriteString(cursor + label + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑↓ navigate · enter rebuild · esc cancel"))
	case 1:
		b.WriteString(accentStyle.Render("Building... this may take a minute") + "\n")
		b.WriteString(subtleStyle.Render("(esc disabled while running)"))
	case 2:
		head := successStyle.Render("✓ Build succeeded")
		if r.isError {
			head = errorStyle.Render("✗ Build failed")
		}
		b.WriteString(head + "\n\n")
		for _, line := range r.output {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("enter back to menu"))
	}

	_ = fmt.Sprintf
	return boxStyle.Render(b.String())
}
