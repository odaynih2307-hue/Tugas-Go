package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/helper"
)

// RequestLogger mencatat setiap HTTP request termasuk identitas user (user_id, role)
// jika request telah melewati middleware autentikasi.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		attrs := []any{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		reqID := c.Get("X-Request-Id")
		if reqID != "" {
			attrs = append(attrs, slog.String("request_id", reqID))
		}

		if user, ok := helper.CurrentUser(c); ok {
			attrs = append(attrs,
				slog.Int("user_id", user.UserID),
				slog.String("role", user.Role),
			)
		}

		logger.Info("http_request", attrs...)
		return err
	}
}
