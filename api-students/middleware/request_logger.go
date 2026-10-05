package middleware

import (
	"errors"
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

		requestID := helper.RequestID(c)

		// Sejak handler mengembalikan error alih-alih menulis response sendiri,
		// status pada c.Response() BELUM terisi ketika baris ini dijalankan:
		// ErrorHandler baru berjalan setelah seluruh rangkaian middleware selesai.
		// Tanpa koreksi di bawah, setiap kegagalan tercatat sebagai 200.
		status := c.Response().StatusCode()
		if err != nil {
			var appErr *helper.AppError
			var fiberErr *fiber.Error
			if errors.As(err, &appErr) {
				status = appErr.Status
			} else if errors.As(err, &fiberErr) {
				status = fiberErr.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		// Catatan Perbaikan Bug Modul Bagian B:
		// Pada modul, baris attrs memakai slog.Int("status", c.Response().StatusCode()),
		// yang selalu mencatat status 200 pada request yang gagal.
		// Kami memperbaikinya menjadi slog.Int("status", status).
		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
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
