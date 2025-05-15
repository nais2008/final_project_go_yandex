package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config calc
type Config struct {
	TimeAdditionMS       int
	TimeSubtractionMS    int
	TimeMultiplicationMS int
	TimeDivisionMS       int
	ComputingPower       int
	AgentAddr            string
	OrchestratorAddr     string
}

// PostgresConfig ...
type PostgresConfig struct {
	DB       string
	User     string
	Password string
	Host     string
	Port     string
}

var envLoaded bool

func loadEnvOnce() {
	if !envLoaded {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file") // Изменили на Println, чтобы не завершать программу
		} else {
			log.Println(".env file loaded successfully")
		}
		envLoaded = true
	}
}

// LoadConfig ...
func LoadConfig() Config {
	loadEnvOnce()

	cfg := Config{
		TimeAdditionMS:       loadEnvInt("TIME_ADDITION_MS", 3000),
		TimeSubtractionMS:    loadEnvInt("TIME_SUBTRACTION_MS", 3000),
		TimeMultiplicationMS: loadEnvInt("TIME_MULTIPLICATIONS_MS", 5000),
		TimeDivisionMS:       loadEnvInt("TIME_DIVISIONS_MS", 5000),
		ComputingPower:       loadEnvInt("COMPUTING_POWER", 4),
		AgentAddr:            loadEnvString("AGENT_ADDR", "localhost:8081"),
		OrchestratorAddr:     loadEnvString("ORCHESTRATOR_ADDR", "localhost:8080"),
	}

	return cfg
}

// LoadPostgresConfig ...
func LoadPostgresConfig() PostgresConfig {
	loadEnvOnce()

	cfg := PostgresConfig{
		DB:       loadEnvString("POSTGRES_DB", "postgres_db"),
		User:     loadEnvString("POSTGRES_USER", "postgres_user"),
		Password: loadEnvString("POSTGRES_PASSWORD", "postgres_password"),
		Host:     loadEnvString("POSTGRES_HOST", "localhost"),
		Port:     loadEnvString("POSTGRES_PORT", "5432"),
	}

	return cfg
}

// loadEnvString ...
func loadEnvString(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// loadEnvInt ...
func loadEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Invalid value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return intVal
}
