package main

import (
	"log"

	"github.com/zenspanel/zenspanel/internal/config"
	"github.com/zenspanel/zenspanel/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := store.New(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Printf("ZensPanel API starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
}
