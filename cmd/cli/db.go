package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/zenspanel/zenspanel/internal/config"
	"github.com/zenspanel/zenspanel/internal/store"
)

// loadConfig wraps config.Load. Kept as a separate helper so future TUI
// code that needs the config doesn't have to know about the underlying
// Viper search-path order.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// connectDB opens the panel database with the same DSN the API uses.
// Returns a *sqlx.DB ready for store package access.
func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	return store.New(cfg.Database.DSN)
}
