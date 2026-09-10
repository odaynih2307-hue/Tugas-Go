package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"api-students/helper"
)

const (
	LocalsUserID   = "user_id"
	LocalsUsername = "username"
	LocalsRole     = "role"
)

func RequireAuth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		if authHeader == "" {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"token akses diperlukan",
			)
		}

		const bearerPrefix = "Bearer "

		if len(authHeader) <= len(bearerPrefix) ||
			authHeader[:len(bearerPrefix)] != bearerPrefix {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"format Authorization tidak valid",
			)
		}

		tokenString := authHeader[len(bearerPrefix):]

		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)

			if errors.Is(err, jwt.ErrTokenExpired) {
				return helper.Fail(
					c,
					fiber.StatusUnauthorized,
					"access token sudah kedaluwarsa",
				)
			}

			return helper.Fail(
				c,
				fiber.StatusUnauthorized,
				"access token tidak valid",
			)
		}

		c.Locals(LocalsUserID, claims.UserID)
		c.Locals(LocalsUsername, claims.Username)
		c.Locals(LocalsRole, claims.Role)

		return c.Next()
	}
}
