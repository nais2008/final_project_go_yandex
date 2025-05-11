package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/utils"
	"github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
	"gorm.io/gorm"
)

type AuthHandler struct {
	gormDB    *gorm.DB
	jwtSecret []byte
}

func NewAuthHandler(gormDB *gorm.DB) *AuthHandler {
	jwtSecret := []byte(os.Getenv("JWT_TOKEN"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_TOKEN environment variable not set")
	}
	return &AuthHandler{
		gormDB:    gormDB,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) RegisterHandler(c echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	email := c.FormValue("email")
	username := c.FormValue("username")
	password := c.FormValue("password")

	log.Printf("Form Values - Email: '%s', Username: '%s', Password: '%s'", email, username, password)

	if email == "" || username == "" || password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "All fields are required")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	user := models.User{
		Email:    email,
		Username: username,
		Password: hashedPassword,
	}

	result := h.gormDB.Create(&user)
	if result.Error != nil {
		log.Printf("Registration failed: %v", result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Registration failed")
	}

	return c.JSON(http.StatusCreated, proto.RegisterResponse{UserId: int64(user.ID)})
}


func (h *AuthHandler) LoginHandler(c echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	login := c.FormValue("login")
	password := c.FormValue("password")

	log.Printf("Login Form Values - Login: '%s', Password: '%s'", login, password)

	if login == "" || password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Login and password are required")
	}

	var user models.User
	result := h.gormDB.Where("username = ? OR email = ?", login, login).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
		}
		log.Printf("Login query failed: %v", result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Login failed")
	}

	if err := utils.ComparePasswords(user.Password, password); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}

	token, err := h.generateJWT(int64(user.ID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate JWT")
	}

	return c.JSON(http.StatusOK, proto.LoginResponse{Token: token})
}

func (h *AuthHandler) generateJWT(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func (h *AuthHandler) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenStr := c.Request().Header.Get("Authorization")
		if tokenStr == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
		}

		parts := strings.Split(tokenStr, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token format")
		}
		tokenStr = parts[1]

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return h.jwtSecret, nil
		})

		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid user ID in token")
			}
			c.Set("user_id", int64(userIDFloat))
			return next(c)
		}

		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}
}
