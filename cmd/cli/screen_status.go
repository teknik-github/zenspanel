package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zenspanel/zenspanel/internal/config"
)

type statusInfo struct {
	services    map[string]string
	dbOK        bool
	dbErr       string
	agentOK     bool
	agentErr    string
	totalUsers  int
	totalDoms   int
	totalDBs    int
	loaded      bool
	statsErr    string
}

type statusModel struct {
	cfg  *config.Config
	info statusInfo
}

func newStatusModel(cfg *config.Config) *statusModel {
	return &statusModel{cfg: cfg}
}

type statusLoadedMsg statusInfo

func (s *statusModel) Init() tea.Cmd {
	return func() tea.Msg { return s.gather() }
}

func (s *statusModel) gather() statusLoadedMsg {
	info := statusInfo{services: map[string]string{}, loaded: true}
	for _, svc := range []string{"zenspanel-api", "zenspanel-agent", "nginx", "mysql", "redis-server"} {
		out, _ := exec.Command("systemctl", "is-active", svc).Output()
		info.services[svc] = strings.TrimSpace(string(out))
	}

	if db, err := connectDB(s.cfg); err != nil {
		info.dbErr = err.Error()
	} else {
		info.dbOK = true
		_ = db.Get(&info.totalUsers, "SELECT COUNT(*) FROM users")
		_ = db.Get(&info.totalDoms, "SELECT COUNT(*) FROM domains")
		_ = db.Get(&info.totalDBs, "SELECT COUNT(*) FROM `databases`")
		db.Close()
	}

	conn, err := net.DialTimeout("unix", s.cfg.Agent.Socket, 2*time.Second)
	if err != nil {
		info.agentErr = err.Error()
	} else {
		info.agentOK = true
		conn.Close()
	}
	return statusLoadedMsg(info)
}

func (m *rootModel) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case statusLoadedMsg:
		m.statusModel.info = statusInfo(v)
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q", "enter":
			return m.backToMenu("", false)
		case "r":
			return m, m.statusModel.Init()
		}
	}
	return m, nil
}

func (m *rootModel) viewStatus() string {
	info := m.statusModel.info
	if !info.loaded {
		return boxStyle.Render(headerStyle.Render("Status & Info") + "\n\nLoading...")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render("Status & Info") + "\n\n")
	b.WriteString(subtleStyle.Render("Services") + "\n")
	for _, svc := range []string{"zenspanel-api", "zenspanel-agent", "nginx", "mysql", "redis-server"} {
		st := info.services[svc]
		mark := errorStyle.Render("✗")
		if st == "active" {
			mark = successStyle.Render("✓")
		}
		b.WriteString(fmt.Sprintf("  %s %-18s %s\n", mark, svc, st))
	}

	b.WriteString("\n" + subtleStyle.Render("Connectivity") + "\n")
	dbMark := errorStyle.Render("✗ failed: " + info.dbErr)
	if info.dbOK {
		dbMark = successStyle.Render("✓ connected")
	}
	agMark := errorStyle.Render("✗ failed: " + info.agentErr)
	if info.agentOK {
		agMark = successStyle.Render("✓ socket reachable")
	}
	b.WriteString("  Database: " + dbMark + "\n")
	b.WriteString("  Agent:    " + agMark + "\n")

	if info.dbOK {
		b.WriteString("\n" + subtleStyle.Render("Panel inventory") + "\n")
		b.WriteString(fmt.Sprintf("  Users:     %d\n", info.totalUsers))
		b.WriteString(fmt.Sprintf("  Domains:   %d\n", info.totalDoms))
		b.WriteString(fmt.Sprintf("  Databases: %d\n", info.totalDBs))
	}

	b.WriteString("\n" + subtleStyle.Render("Config: "+m.cfg.Database.DSN) + "\n")
	b.WriteString(helpStyle.Render("r refresh · esc/q back"))
	return boxStyle.Render(b.String())
}
