package main

import (
    "log/slog"

    "github.com/nais2008/final_project_go_yandex/internal/agent"
    "github.com/nais2008/final_project_go_yandex/internal/config"
)

func main() {
    const op = "main.Agent"

    cfg := config.LoadConfig()
    ag := agent.NewAgent(cfg)
    if ag == nil {
        slog.Error(op, "failed to create agent")
        return
    }

    for i := 0; i < cfg.ComputingPower; i++ {
        go ag.Run()
    }

    slog.Info(op, "Agent started", "workers", cfg.ComputingPower)
    select {}
}
