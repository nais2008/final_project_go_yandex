package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/nais2008/final_project_go_yandex/internal/auth"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/orchestrator"
	"github.com/nais2008/final_project_go_yandex/internal/renderer"
	customMiddleware "github.com/nais2008/final_project_go_yandex/internal/middleware"
)

func main() {
	cfg := config.LoadConfig()

	storage, err := db.ConnectDB()
	if err != nil {
		panic("not db connetct")
	}

	e := echo.New()

	e.Renderer = renderer.NewRenderer("templates/*.html")
	e.Static("/static", "../../static")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())


	orch := orchestrator.NewOrchestrator(cfg, storage)

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.POST("/api/v1/register", auth.RegisterUser(storage))
	e.POST("/api/v1/login", auth.LoginUser(storage))

	api := e.Group("/api/v1")
	api.Use(customMiddleware.AuthMiddleware(storage))
	api.POST("/calculate", orch.CalculateHandler)
	api.GET("/expressions", orch.GetExpressionsHandler)
	api.GET("/expressions/:id", orch.GetExpressionByIDHandler)

	internal := e.Group("/internal")
	internal.GET("/tasks", orch.TaskHandler)
	internal.POST("/tasks", orch.TaskHandler)

	log.Printf("Orchestrator listening on %s", cfg.OrchestratorAddr)
	log.Fatal(e.Start(cfg.OrchestratorAddr))
}
