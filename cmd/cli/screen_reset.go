package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/zenspanel/zenspanel/internal/store"
)

type resetPasswordModel struct {
	username textinput.Model
	password textinput.Model
	focus    int // 0 = username, 1 = password
	message  string
	isError  bool
	done     bool
}

func newResetPasswordModel() *resetPasswordModel {
	u := textinput.New()
	u.Placeholder = "username"
	u.Focus()
	u.CharLimit = 64

	p := textinput.New()
	p.Placeholder = "new password (min 8 chars)"
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '•'
	p.CharLimit = 128

	return &resetPasswordModel{username: u, password: p}
}

func (m *rootModel) updateReset(msg tea.Msg) (tea.Model, tea.Cmd) {
	r := m.resetModel
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "esc":
			return m.backToMenu("", false)
		case "tab", "shift+tab":
			r.focus = 1 - r.focus
			if r.focus == 0 {
				r.username.Focus()
				r.password.Blur()
			} else {
				r.username.Blur()
				r.password.Focus()
			}
			return m, nil
		case "enter":
			if r.done {
				return m.backToMenu(r.message, r.isError)
			}
			if r.focus == 0 && strings.TrimSpace(r.username.Value()) != "" {
				r.focus = 1
				r.username.Blur()
				r.password.Focus()
				return m, nil
			}
			if err := r.execute(); err != nil {
				r.message = err.Error()
				r.isError = true
			} else {
				r.message = fmt.Sprintf("Password updated for %s", r.username.Value())
				r.isError = false
			}
			r.done = true
			return m, nil
		}
	}
	var cmd tea.Cmd
	if r.focus == 0 {
		r.username, cmd = r.username.Update(msg)
	} else {
		r.password, cmd = r.password.Update(msg)
	}
	return m, cmd
}

func (r *resetPasswordModel) execute() error {
	user := strings.TrimSpace(r.username.Value())
	pass := r.password.Value()
	if len(pass) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
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
	hash, err := store.HashPassword(pass)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	// Bypass the dynamic Update allowlist (which excludes password_hash on
	// purpose for HTTP callers) by going straight at the table — the CLI
	// is a trusted local operator, not a request from over the network.
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, u.ID)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (m *rootModel) viewReset() string {
	r := m.resetModel
	var b strings.Builder
	b.WriteString(headerStyle.Render("Reset Password") + "\n\n")
	b.WriteString(subtleStyle.Render("Username") + "\n")
	b.WriteString(r.username.View() + "\n\n")
	b.WriteString(subtleStyle.Render("New password") + "\n")
	b.WriteString(r.password.View() + "\n")

	if r.message != "" {
		b.WriteString("\n")
		if r.isError {
			b.WriteString(errorStyle.Render(r.message))
		} else {
			b.WriteString(successStyle.Render(r.message))
		}
		b.WriteString("\n")
	}

	if r.done {
		b.WriteString(helpStyle.Render("enter back to menu"))
	} else {
		b.WriteString(helpStyle.Render("tab switch · enter submit · esc cancel"))
	}
	return boxStyle.Render(b.String())
}
