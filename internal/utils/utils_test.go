package utils_test

import (
	"os"
	"testing"

	"github.com/nais2008/final_project_go_yandex/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	hash, err := utils.HashPassword("password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestComparePasswords(t *testing.T) {
	hash, _ := utils.HashPassword("password123")
	err := utils.ComparePasswords(hash, "password123")
	assert.NoError(t, err)

	err = utils.ComparePasswords(hash, "wrongpassword")
	assert.Error(t, err)
}

// TestGenerateJWT проверяет создание JWT токена.
func TestGenerateJWT(t *testing.T) {
	os.Setenv("JWT_TOKEN", "test_secret")
	token, err := utils.GenerateJWT("testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestVerifyJWT_Valid проверяет успешную проверку JWT.
func TestVerifyJWT_Valid(t *testing.T) {
	os.Setenv("JWT_TOKEN", "test_secret")
	token, err := utils.GenerateJWT("testuser")
	assert.NoError(t, err)

	claims, err := utils.VerifyJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", claims.Login)
}

// TestVerifyJWT_Invalid проверяет ошибку при неправильном JWT.
func TestVerifyJWT_Invalid(t *testing.T) {
	os.Setenv("JWT_TOKEN", "test_secret")
	_, err := utils.VerifyJWT("invalid_token")
	assert.Error(t, err)
}
