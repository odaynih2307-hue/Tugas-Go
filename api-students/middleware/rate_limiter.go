package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"api-students/helper"
)

func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,

		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},

		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")

			return helper.Fail(
				c,
				fiber.StatusTooManyRequests,
				"terlalu banyak percobaan login, coba lagi dalam satu menit",
			)
		},
	})
}
