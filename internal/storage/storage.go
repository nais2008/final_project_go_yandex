package storage

import "errors"

var (
	ErrUserExist = errors.New("user already exist")
	ErrUserNotFound = errors.New("user not found")
	ErrExpressionNotFound = errors.New("expression not found")
	ErrTaskNotFound = errors.New("task not found")
)

