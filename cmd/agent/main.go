package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/nais2008/final_project_go_yandex/internal/agent"
	"github.com/nais2008/final_project_go_yandex/internal/config"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	cfg := config.LoadConfig() // Используем вашу функцию LoadConfig

	// Вы можете добавить логирование конфигурации для отладки
	log.Printf("Agent Configuration: %+v", cfg)

	// Создаем экземпляр агента
	a := agent.NewAgent(cfg)

	// Запускаем агента, используя cfg.GrpcServerAddr для подключения к оркестратору по gRPC
	log.Printf("Agent starting. Connecting to gRPC server at: %s", cfg.GrpcServerAddr)
	a.Run()

	// Агент будет работать в бесконечном цикле
}
