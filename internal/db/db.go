package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
)

type Storage struct{
	db *gorm.DB
}

// ConnectDB connected database(psql)
func ConnectDB() (*Storage, error) {
	const op string = "db.ConnectDB"

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
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Expression{},
		&models.Storage{},
		&models.Task{},
	); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}
