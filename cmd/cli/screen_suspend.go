package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/zenspanel/zenspanel/internal/store"
)

type suspendModel struct {
	username textinput.Model
	choice   int // 0 = suspend, 1 = unsuspend
	stage    int // 0 = enter username, 1 = pick action, 2 = done
	message  string
	isError  bool
}

func newSuspendModel() *suspendModel {
	u := textinput.New()
	u.Placeholder = "username"
	u.CharLimit = 64
	u.Focus()
	return &suspendModel{username: u}
}

func (m *rootModel) updateSuspend(msg tea.Msg) (tea.Model, tea.Cmd) {
	s := m.suspendModel
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "esc":
			return m.backToMenu("", false)
		case "enter":
			if s.stage == 2 {
				return m.backToMenu(s.message, s.isError)
			}
			if s.stage == 0 {
				if strings.TrimSpace(s.username.Value()) == "" {
					return m, nil
				}
				s.stage = 1
				return m, nil
			}
			// stage 1: execute
			if err := s.execute(); err != nil {
				s.message = err.Error()
				s.isError = true
			} else {
				action := "suspended"
				if s.choice == 1 {
					action = "unsuspended"
				}
				s.message = fmt.Sprintf("User %q %s", s.username.Value(), action)
				s.isError = false
			}
			s.stage = 2
			return m, nil
		case "left", "h":
			if s.stage == 1 {
				s.choice = 0
			}
		case "right", "l":
			if s.stage == 1 {
				s.choice = 1
			}
		case "tab":
			if s.stage == 1 {
				s.choice = 1 - s.choice
			}
		}
	}
	if s.stage == 0 {
		var cmd tea.Cmd
		s.username, cmd = s.username.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (s *suspendModel) execute() error {
	user := strings.TrimSpace(s.username.Value())
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	db, err := connectDB(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer db.Close()
	users := store.NewUserStore(db)
	u, err := users.GetByUsername(user)
	if err != nil {
		return fmt.Errorf("user %q not found", user)
	}
	status := "suspended"
	if s.choice == 1 {
		status = "active"
	}
	return users.Update(u.ID, map[string]interface{}{"status": status})
}

func (m *rootModel) viewSuspend() string {
	s := m.suspendModel
	var b strings.Builder
	b.WriteString(headerStyle.Render("Suspend / Unsuspend User") + "\n\n")
	b.WriteString(subtleStyle.Render("Username") + "\n")
	b.WriteString(s.username.View() + "\n\n")

	if s.stage >= 1 {
		opts := []string{"Suspend", "Unsuspend"}
		b.WriteString(subtleStyle.Render("Action") + "\n  ")
		for i, o := range opts {
			if i == s.choice {
				b.WriteString(selectedStyle.Render("[ "+o+" ]"))
			} else {
				b.WriteString(subtleStyle.Render("[ " + o + " ]"))
			}
			b.WriteString("  ")
		}
		b.WriteString("\n")
	}
	if s.message != "" {
		b.WriteString("\n")
		if s.isError {
			b.WriteString(errorStyle.Render(s.message))
		} else {
			b.WriteString(successStyle.Render(s.message))
		}
		b.WriteString("\n")
	}
	if s.stage == 2 {
		b.WriteString(helpStyle.Render("enter back to menu"))
	} else if s.stage == 1 {
		b.WriteString(helpStyle.Render("←→ select · enter confirm · esc cancel"))
	} else {
		b.WriteString(helpStyle.Render("enter next · esc cancel"))
	}
	return boxStyle.Render(b.String())
}
