package main

import (
	"log"
	"github.com/zenspanel/zenspanel/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("ZensPanel Agent starting, socket: %s", cfg.Agent.Socket)
}
