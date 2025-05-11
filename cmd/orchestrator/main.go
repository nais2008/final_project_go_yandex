package main

import (
	"log"
	"net"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/orchestrator"
	"github.com/nais2008/final_project_go_yandex/internal/renderer"
	proto "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
)

func main() {
	cfg := config.LoadConfig()
	dbConn := db.ConnectDB()

	authConn, err := grpc.Dial(cfg.GrpcServerAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to auth service: %v", err)
	}
	defer authConn.Close()

	orch := orchestrator.NewOrchestrator(cfg, dbConn, authConn)

	lis, err := net.Listen("tcp", cfg.OrchestratorAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterOrchestratorServiceServer(s, orch)

	go func() {
		log.Printf("Starting gRPC orchestrator on %s", cfg.OrchestratorAddr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	e := echo.New()
	e.Renderer = renderer.NewRenderer("templates/*.html")
	e.Static("/static", "static")

	protected := e.Group("", orch.AuthMiddleware())

	protected.POST("/api/v1/calculate", orch.CalculateHandler)
	protected.GET("/api/v1/expressions", orch.GetExpressionsHandler)
	protected.GET("/api/v1/expressions/:id", orch.GetExpressionByIDHandler)
	protected.GET("/internal/task", orch.TaskGetHandler)
	protected.POST("/internal/task", orch.TaskPostHandler)

	e.POST("/api/v1/register", orch.RegisterHandler)
	e.POST("/api/v1/login", orch.LoginHandler)

	e.GET("/", orch.IndexHandler)

	log.Printf("Starting HTTP server on :8080")
	if err := e.Start(":8080"); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
