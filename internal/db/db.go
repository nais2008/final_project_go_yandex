package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
)

// ConnectDB connected database(psql)
func ConnectDB() *gorm.DB {
	cfg := config.LoadPostgresConfig()

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
	)

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Expression{},
		&models.Storage{},
		&models.Task{},
	); err != nil {
		log.Fatalf("Error migrate database: %v", err)
	}

	return db
}
