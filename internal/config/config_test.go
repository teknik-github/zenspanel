package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Agent.Socket != "/run/zenspanel/agent.sock" {
		t.Errorf("unexpected socket: %s", cfg.Agent.Socket)
	}
	if cfg.Paths.HomeBase != "/home/zenspanel" {
		t.Errorf("unexpected home_base: %s", cfg.Paths.HomeBase)
	}
}
