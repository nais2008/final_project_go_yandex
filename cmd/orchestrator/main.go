package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/orchestrator"
	"github.com/nais2008/final_project_go_yandex/internal/renderer"
	"github.com/nais2008/final_project_go_yandex/proto"
	"google.golang.org/grpc"
)

func grpcOrHTTP(grpcSrv *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcSrv.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}

func main() {
	cfgCalc := config.LoadConfig()
	pgCfg := config.LoadPostgresConfig()
	gormDB := db.ConnectDB(pgCfg)
	orcSvc := orchestrator.NewOrchestrator(gormDB, cfgCalc)

	grpcSrv := grpc.NewServer()
	proto.RegisterOrchestratorServiceServer(grpcSrv, orcSvc)
	proto.RegisterAgentServiceServer(grpcSrv, orcSvc)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())

	rt := renderer.NewRenderer("templates")
	e.Renderer = rt

	e.Static("/static", "static")


	e.GET("/", func(c echo.Context) error {
		// При рендере можно передать CSRF и/или токен, сейчас — пусто
		return c.Render(http.StatusOK, "base.html", map[string]interface{}{})
	})

	jwtSecret := os.Getenv("JWT_TOKEN")
	e.POST("/api/v1/auth/register", func(c echo.Context) error {
		var req proto.RegisterRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"user_id": 0})
	})
	e.POST("/api/v1/auth/login", func(c echo.Context) error {
		var req proto.LoginRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"token": jwtSecret})
	})

	api := e.Group("/api/v1", middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(jwtSecret),
	}))
	api.POST("/expressions", func(c echo.Context) error {
		var body struct{ Expression string }
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		resp, err := orcSvc.SubmitExpression(c.Request().Context(), &proto.ExpressionRequest{
			Expression: body.Expression,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusCreated, resp)
	})
	api.GET("/expressions", func(c echo.Context) error {
		exprs := []orchestrator.ExpressionWithTasks{}
		orcSvc.ListHTTP(c, &exprs)
		return c.JSON(http.StatusOK, exprs)
	})
	api.GET("/expressions/:id", func(c echo.Context) error {
		var expr orchestrator.ExpressionWithTasks
		orcSvc.GetByIDHTTP(c, &expr)
		return c.JSON(http.StatusOK, expr)
	})

	// Единый сервер: грэйсфул shutdown опущен
	handler := grpcOrHTTP(grpcSrv, e)
	lis, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatalf("listen :80 failed: %v", err)
	}
	log.Println("serving gRPC+HTTP on :80")
	if err := http.Serve(lis, handler); err != nil {
		log.Fatalf("serve error: %v", err)
	}
}
