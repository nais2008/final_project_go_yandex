package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/utils"
)

// AuthMiddleware ...
func AuthMiddleware(storage *db.Storage) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := utils.VerifyJWT(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
			}

			user, err := storage.User(c.Request().Context(), claims.Login)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid user"})
			}

			c.Set("user_id", user.ID)
			c.Set("username", claims.Login)
			return next(c)
		}
	}
}
