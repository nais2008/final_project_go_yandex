package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword ...
func HashPassword(password string) ([]byte, error) {
	const op string = "utils.HashPassword"

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return hashedPassword, nil
}

// ComparePasswords ...
func ComparePasswords(hashedPassword []byte, password string) error {
	return bcrypt.CompareHashAndPassword(
		hashedPassword,
		[]byte(password),
	)
}
