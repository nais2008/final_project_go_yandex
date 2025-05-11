package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/nais2008/final_project_go_yandex/proto"

	"google.golang.org/grpc"
)

func main() {
	// Загружаем конфиг
	cfgCalc := config.LoadConfig()
	pgCfg   := config.LoadPostgresConfig()
	dbConn  := db.ConnectDB(pgCfg)
	orc     := orchestrator.NewOrchestrator(dbConn, cfgCalc)

	// 1) Запускаем gRPC-сервер Оркестратора на 50051
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("gRPC listen failed: %v", err)
		}
		grpcSrv := grpc.NewServer()
		proto.RegisterOrchestratorServiceServer(grpcSrv, orc)
		log.Println("gRPC server running on :50051")
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC serve failed: %v", err)
		}
	}()

	// 2) HTTP-API на порту 80
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())

	jwtSecret := os.Getenv("JWT_TOKEN")
	e.POST("/api/v1/auth/register", func(c echo.Context) error {
		var req proto.RegisterRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		// ваша логика регистрации...
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
		resp, err := orc.SubmitExpression(c.Request().Context(), &proto.ExpressionRequest{
			Expression: body.Expression,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		// вернём ID выражения
		var out struct{ ID uint64 }
		out.ID, _ = strconv.ParseUint(resp.Result, 10, 64)
		return c.JSON(http.StatusCreated, out)
	})
	api.GET("/expressions", func(c echo.Context) error {
		list, err := orc.ListExpressionsHTTP(c.Request().Context())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, list)
	})
	api.GET("/expressions/:id", func(c echo.Context) error {
		id := c.Param("id")
		expr, err := orc.GetExpressionByIDHTTP(c.Request().Context(), id)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return c.JSON(http.StatusOK, expr)
	})

	log.Println("HTTP server running on :80")
	log.Fatal(e.Start(":80"))
}
