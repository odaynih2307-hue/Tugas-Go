package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/helper"
)

func RequireJSON(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")

	if !strings.HasPrefix(contentType, "application/json") {
		return helper.Fail(
			c,
			fiber.StatusUnsupportedMediaType,
			"Content-Type harus application/json",
		)
	}

	return c.Next()
}
