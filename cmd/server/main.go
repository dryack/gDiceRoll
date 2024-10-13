package main

import (
	"log"

	"github.com/dryack/gDiceRoll/core/api"
	"github.com/dryack/gDiceRoll/core/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting server on %s", cfg.String("server.address"))
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
