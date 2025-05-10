package auth

import (
	"fmt"
	"log"
	"time"

	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/utils"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtSecretKey string

// SetJWTSecretKey ...
func SetJWTSecretKey(key string) {
	jwtSecretKey = key
}

// GenerateJWT ...
func GenerateJWT(userID uint) (string, error) {
	if jwtSecretKey == "" {
		log.Fatal("JWT_TOKEN не установлен")
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = userID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// RegisterUser ...
func RegisterUser(db *gorm.DB, username, email, password string) (*models.User, error) {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("ошибка хэширования пароля: %w", err)
	}
	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}

	if err := db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("ошибка сохранения пользователя: %w", err)
	}

	return user, nil
}

// LoginUser ...
func LoginUser(db *gorm.DB, identifier, password string) (string, error) {
	var user models.User
	if err := db.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error; err != nil {
		return "", fmt.Errorf("пользователь не найден: %w", err)
	}

	if !password.CheckPassword(user.Password, password) {
		return "", fmt.Errorf("неправильный пароль")
	}

	token, err := GenerateJWT(user.ID)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации токена: %w", err)
	}

	return token, nil
}
