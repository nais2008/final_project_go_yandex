package storage

import "errors"

var (
	// ErrUserExist ...
	ErrUserExist = errors.New("user already exist")
	// ErrUserNotFound ...
	ErrUserNotFound = errors.New("user not found")
	// ErrExpressionNotFound ...
	ErrExpressionNotFound = errors.New("expression not found")
	// ErrTaskNotFound ...
	ErrTaskNotFound = errors.New("task not found")
)

