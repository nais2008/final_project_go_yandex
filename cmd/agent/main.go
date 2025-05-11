package main

import (
	"log"
	"os"

	"github.com/nais2008/final_project_go_yandex/internal/agent"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig()

	target := os.Getenv("AGENT_URL")
	if target == "" {
		log.Fatal("AGENT_URL must be set, e.g. AGENT_URL=localhost:50051")
	}

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial Orchestrator gRPC (%s): %v", target, err)
	}
	defer conn.Close()

	ag, err := agent.NewAgent(conn, cfg)
	if err != nil {
		log.Fatalf("NewAgent failed: %v", err)
	}
	defer ag.Close()

	ag.Run()
	select {}
}
