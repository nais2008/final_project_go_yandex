package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/storage"
)

var (
	ErrUserEmailExists  = errors.New("user with this email already exists")
	ErrUsernameExists   = errors.New("user with this username already exists")
)

// SaveUser ...
func (s *Storage) SaveUser(
	ctx context.Context,
	username string,
	email string,
	passHash []byte,
) (int64, error) {
	const op string = "db.SaveUser"

	user := models.User{
		Email: email,
		Username: username,
		Password: passHash,
	}

	res := s.DB.WithContext(ctx).Create(&user)

	if res.Error != nil {
		if isDuplicateError(res.Error, "email") {
			return 0, fmt.Errorf("%s: %w", op, ErrUserEmailExists)
		}
		if isDuplicateError(res.Error, "username") {
			return 0, fmt.Errorf("%s: %w", op, ErrUsernameExists)
		}
		return 0, fmt.Errorf("%s: %w", op, res.Error)
	}

	return int64(user.ID), nil
}

func isDuplicateError(err error, field string) bool {
	return err != nil &&
		(strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
			strings.Contains(strings.ToLower(err.Error()), "unique")) &&
		strings.Contains(strings.ToLower(err.Error()), field)
}

// User ...
func (s *Storage) User(
	ctx context.Context,
	login string,
) (models.User, error){
	const op string = "db.User"

	var user models.User
	err := s.DB.WithContext(ctx).Where(
		"email = ? OR username = ?", login, login,
	).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrExpressionNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
