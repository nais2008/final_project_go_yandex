package main

import (
	"log"

	"github.com/nais2008/final_project_go_yandex/internal/agent"
	"github.com/nais2008/final_project_go_yandex/internal/config"
)

func main() {
	cfg := config.LoadConfig()

	ag := agent.NewAgent(cfg)

	log.Printf("Starting agent, connecting to %s", cfg.GrpcServerAddr)
	if err := ag.Run(); err != nil {
		log.Fatalf("agent failed: %v", err)
	}
}
