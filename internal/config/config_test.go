package config_test

import (
	"os"
	"testing"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/stretchr/testify/assert"
)


func TestLoadConfig_WithEnvVariables(t *testing.T) {
	os.Setenv("TIME_ADDITION_MS", "1000")
	os.Setenv("TIME_SUBTRACTION_MS", "2000")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "3000")
	os.Setenv("TIME_DIVISIONS_MS", "4000")
	os.Setenv("COMPUTING_POWER", "8")
	os.Setenv("AGENT_ADDR", "agent.example.com:8082")
	os.Setenv("ORCHESTRATOR_ADDR", "orch.example.com:8081")

	defer os.Unsetenv("TIME_ADDITION_MS")
	defer os.Unsetenv("TIME_SUBTRACTION_MS")
	defer os.Unsetenv("TIME_MULTIPLICATIONS_MS")
	defer os.Unsetenv("TIME_DIVISIONS_MS")
	defer os.Unsetenv("COMPUTING_POWER")
	defer os.Unsetenv("AGENT_ADDR")
	defer os.Unsetenv("ORCHESTRATOR_ADDR")

	cfg := config.LoadConfig()

	assert.Equal(t, 1000, cfg.TimeAdditionMS)
	assert.Equal(t, 2000, cfg.TimeSubtractionMS)
	assert.Equal(t, 3000, cfg.TimeMultiplicationMS)
	assert.Equal(t, 4000, cfg.TimeDivisionMS)
	assert.Equal(t, 8, cfg.ComputingPower)
	assert.Equal(t, "agent.example.com:8082", cfg.AgentAddr)
	assert.Equal(t, "orch.example.com:8081", cfg.OrchestratorAddr)
}

func TestLoadConfig_WithDefaultValues(t *testing.T) {

	cfg := config.LoadConfig()

	assert.Equal(t, 3000, cfg.TimeAdditionMS)
	assert.Equal(t, 3000, cfg.TimeSubtractionMS)
	assert.Equal(t, 5000, cfg.TimeMultiplicationMS)
	assert.Equal(t, 5000, cfg.TimeDivisionMS)
	assert.Equal(t, 4, cfg.ComputingPower)
	assert.Equal(t, "localhost:8081", cfg.AgentAddr)
	assert.Equal(t, "localhost:8080", cfg.OrchestratorAddr)
}

func TestLoadConfig_InvalidIntValue(t *testing.T) {
	os.Setenv("COMPUTING_POWER", "invalid")
	defer os.Unsetenv("COMPUTING_POWER")

	cfg := config.LoadConfig()

	assert.Equal(t, 4, cfg.ComputingPower)
}

func TestLoadPostgresConfig_WithEnvVariables(t *testing.T) {
	os.Setenv("POSTGRES_DB", "test_db")
	os.Setenv("POSTGRES_USER", "test_user")
	os.Setenv("POSTGRES_PASSWORD", "test_password")
	os.Setenv("POSTGRES_HOST", "test.host")
	os.Setenv("POSTGRES_PORT", "5433")

	defer os.Unsetenv("POSTGRES_DB")
	defer os.Unsetenv("POSTGRES_USER")
	defer os.Unsetenv("POSTGRES_PASSWORD")
	defer os.Unsetenv("POSTGRES_HOST")
	defer os.Unsetenv("POSTGRES_PORT")

	cfg := config.LoadPostgresConfig()

	assert.Equal(t, "test_db", cfg.DB)
	assert.Equal(t, "test_user", cfg.User)
	assert.Equal(t, "test_password", cfg.Password)
	assert.Equal(t, "test.host", cfg.Host)
	assert.Equal(t, "5433", cfg.Port)
}

func TestLoadPostgresConfig_WithDefaultValues(t *testing.T) {

	cfg := config.LoadPostgresConfig()

	assert.Equal(t, "postgres_db", cfg.DB)
	assert.Equal(t, "postgres_user", cfg.User)
	assert.Equal(t, "postgres_password", cfg.Password)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
}
