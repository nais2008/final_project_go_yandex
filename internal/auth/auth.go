package auth

import (
	"net/http"

	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/utils"

	"github.com/labstack/echo/v4"
)

// LoginRequest ...
type LoginRequest struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest ...
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginUser ...
func LoginUser(storage *db.Storage) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req LoginRequest

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
		}

		user, err := storage.User(c.Request().Context(), req.Login)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid login or password"})
		}

		err = utils.ComparePasswords(user.Password, req.Password)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid login or password"})
		}

		token, err := utils.GenerateJWT(user.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error generating token"})
		}

		return c.JSON(http.StatusOK, map[string]string{"token": token})
	}
}

// RegisterUser ...
func RegisterUser(storage *db.Storage) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req RegisterRequest

		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format: " + err.Error(),
			})
		}

		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error hashing password"})
		}

		_, err = storage.SaveUser(c.Request().Context(), req.Username, req.Email, hash)
		if err != nil {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusCreated, map[string]string{"message": "User registered successfully"})
	}
}

