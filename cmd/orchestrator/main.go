package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/auth"
	"github.com/nais2008/final_project_go_yandex/internal/renderer"
)

func main() {
	cfg := config.LoadConfig()

	storage, err := db.ConnectDB()
	if err != nil {
		panic("not db connetct")
	}

	e := echo.New()

	// base settings
	e.Renderer = renderer.NewRenderer("templates/*.html")
	e.Static("/static", "../../static")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())


	// orch := orchestrator.NewOrchestrator(cfg, gormDB)

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.POST("/api/v1/register", auth.RegisterUser(storage))
	e.POST("/api/v1/login", auth.LoginUser(storage))

	// api := e.Group("/api/v1")
	// api.Use(authHandler.AuthMiddleware)
	// api.POST("/calculate", orch.CalculateHandler)
	// api.GET("/expressions", orch.GetExpressionsHandler)
	// api.GET("/expressions/:id", orch.GetExpressionByIDHandler)

	// internal := e.Group("/api/v1/internal")
	// internal.Use(authHandler.AuthMiddleware)
	// internal.GET("/task", orch.TaskGetHandler)
	// internal.POST("/task", orch.TaskPostHandler)

	log.Printf("Orchestrator listening on %s", cfg.OrchestratorAddr)
	log.Fatal(e.Start(cfg.OrchestratorAddr))
}
