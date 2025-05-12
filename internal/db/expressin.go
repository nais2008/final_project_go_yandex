package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/storage"
)

var (
	ErrExpressionNotFound = errors.New("expression not found")
)

// SaveExpression ...
func (s *Storage) SaveExpression(
    ctx context.Context,
    expression string,
    userID uint,
    storageID uint,
) (int64, error) {
    const op string = "db.SaveExpression"

    expr := models.Expression{
        Expr:       expression,
        Status:     "pending",
        UserID:     userID,
        StorageID:  storageID,
    }

    res := s.db.WithContext(ctx).Create(&expr)

    if res.Error != nil {
        return 0, fmt.Errorf("%s: %w", op, res.Error)
    }

    return int64(expr.ID), nil
}

// Expressions ...
func (s *Storage) Expressions(
    ctx context.Context,
) ([]models.Expression, error) {
    const op string = "db.Expressions"

    var expressions []models.Expression
    err := s.db.WithContext(ctx).Find(&expressions).Error
    if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }

    return expressions, nil
}

// ExpressionByID ...
func (s *Storage) ExpressionByID(
    ctx context.Context,
    id uint,
) (models.Expression, error) {
    const op string = "db.ExpressionByID"

    var expression models.Expression
    err := s.db.WithContext(ctx).Where("id = ?", id).First(&expression).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return models.Expression{}, fmt.Errorf("%s: %w", op, storage.ErrExpressionNotFound)
        }
        return models.Expression{}, fmt.Errorf("%s: %w", op, err)
    }

    return expression, nil
}

// Task ...
func (s *Storage) Task(
    ctx context.Context,
) (models.Task, error) {
    const op string = "db.GetTask"

    var task models.Task
    err := s.db.WithContext(ctx).Where("status = ?", "pending").First(&task).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return models.Task{}, fmt.Errorf("%s: %w", op, storage.ErrTaskNotFound)
        }
        return models.Task{}, fmt.Errorf("%s: %w", op, err)
    }

    return task, nil
}

// SubmitResult ...
func (s *Storage) SubmitResult(
    ctx context.Context,
    taskID uint,
    result float64,
) error {
    const op string = "db.SubmitResult"

    task := models.Task{
        ID:     taskID,
        Result: &result,
        Status: "completed",
    }

    res := s.db.WithContext(ctx).Save(&task)
    if res.Error != nil {
        return fmt.Errorf("%s: %w", op, res.Error)
    }

    return nil
}
