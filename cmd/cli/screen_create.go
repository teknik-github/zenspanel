package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/zenspanel/zenspanel/internal/api/handlers"
	"github.com/zenspanel/zenspanel/internal/store"
)

type createAdminModel struct {
	inputs  []textinput.Model
	focus   int
	message string
	isError bool
	done    bool
}

func newCreateAdminModel() *createAdminModel {
	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 64
	username.Focus()

	email := textinput.New()
	email.Placeholder = "email@example.com"
	email.CharLimit = 255

	password := textinput.New()
	password.Placeholder = "password (min 8 chars)"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 128

	return &createAdminModel{inputs: []textinput.Model{username, email, password}}
}

func (m *rootModel) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	c := m.createModel
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "esc":
			return m.backToMenu("", false)
		case "tab", "down":
			c.focusNext(1)
			return m, nil
		case "shift+tab", "up":
			c.focusNext(-1)
			return m, nil
		case "enter":
			if c.done {
				return m.backToMenu(c.message, c.isError)
			}
			if c.focus < len(c.inputs)-1 {
				c.focusNext(1)
				return m, nil
			}
			if err := c.execute(); err != nil {
				c.message = err.Error()
				c.isError = true
			} else {
				c.message = fmt.Sprintf("Admin %q created", c.inputs[0].Value())
				c.isError = false
			}
			c.done = true
			return m, nil
		}
	}
	var cmd tea.Cmd
	c.inputs[c.focus], cmd = c.inputs[c.focus].Update(msg)
	return m, cmd
}

func (c *createAdminModel) focusNext(delta int) {
	c.inputs[c.focus].Blur()
	c.focus = (c.focus + delta + len(c.inputs)) % len(c.inputs)
	c.inputs[c.focus].Focus()
}

func (c *createAdminModel) execute() error {
	username := strings.TrimSpace(c.inputs[0].Value())
	email := strings.TrimSpace(c.inputs[1].Value())
	pass := c.inputs[2].Value()

	if err := handlers.ValidateUsername(username); err != nil {
		return err
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}
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

	hash, err := store.HashPassword(pass)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	maxUID, _ := users.GetMaxLinuxUID()
	user := &store.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         "admin",
		LinuxUID:     maxUID + 1,
		Status:       "active",
	}
	if err := users.Create(user); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	return nil
}

func (m *rootModel) viewCreate() string {
	c := m.createModel
	labels := []string{"Username", "Email", "Password"}
	var b strings.Builder
	b.WriteString(headerStyle.Render("Create Admin") + "\n\n")
	for i, in := range c.inputs {
		b.WriteString(subtleStyle.Render(labels[i]) + "\n")
		b.WriteString(in.View() + "\n\n")
	}
	if c.message != "" {
		if c.isError {
			b.WriteString(errorStyle.Render(c.message))
		} else {
			b.WriteString(successStyle.Render(c.message))
		}
		b.WriteString("\n")
	}
	if c.done {
		b.WriteString(helpStyle.Render("enter back to menu"))
	} else {
		b.WriteString(helpStyle.Render("tab/↑↓ switch · enter submit · esc cancel"))
	}
	return boxStyle.Render(b.String())
}
