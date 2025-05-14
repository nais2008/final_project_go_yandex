package main

import (
	"log"

	"github.com/nais2008/final_project_go_yandex/internal/agent"
	"github.com/nais2008/final_project_go_yandex/internal/config"
)

func main() {
    cfg := config.LoadConfig()
    computingPower := cfg.ComputingPower

    ag := agent.NewAgent(cfg)
    for i := 0; i < computingPower; i++ {
        go ag.Run()
    }

    log.Printf("Agent started with %d workers", computingPower)
    select {}
}
