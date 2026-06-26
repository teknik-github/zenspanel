package config

import (
	"strings"
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

func TestValidateRejectsWeakJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid production secret",
			cfg: Config{
				JWT:      JWTConfig{Secret: strings.Repeat("a", 32)},
				Database: DatabaseConfig{DSN: "user:pass@tcp(127.0.0.1:3306)/zenspanel"},
			},
			wantErr: false,
		},
		{
			name: "default dev secret",
			cfg: Config{
				JWT:      JWTConfig{Secret: "dev-secret-change-in-production"},
				Database: DatabaseConfig{DSN: "user:pass@tcp(127.0.0.1:3306)/zenspanel"},
			},
			wantErr: true,
		},
		{
			name: "short attacker guessable secret",
			cfg: Config{
				JWT:      JWTConfig{Secret: "secret"},
				Database: DatabaseConfig{DSN: "user:pass@tcp(127.0.0.1:3306)/zenspanel"},
			},
			wantErr: true,
		},
		{
			name: "missing database dsn",
			cfg: Config{
				JWT: JWTConfig{Secret: strings.Repeat("b", 32)},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
