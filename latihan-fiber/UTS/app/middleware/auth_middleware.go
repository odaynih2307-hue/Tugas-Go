package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/helper"
	"uts-siakad/app/repository"
)

// Authenticate memvalidasi JWT Bearer token pada header Authorization.
// Sesuai PDF: Mahasiswa yang di-soft delete tidak dapat login dan tokennya tidak sah.
func Authenticate(userRepo repository.UserRepository, studentRepo repository.StudentRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return helper.Error(c, fiber.StatusUnauthorized, "Token autentikasi tidak ditemukan")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return helper.Error(c, fiber.StatusUnauthorized, "Format header Authorization harus Bearer <token>")
		}

		tokenStr := parts[1]
		claims, err := helper.ValidateToken(tokenStr)
		if err != nil {
			return helper.Error(c, fiber.StatusUnauthorized, "Token tidak valid atau telah kedaluwarsa")
		}

		// Pastikan user masih ada di basis data
		user, err := userRepo.FindByID(c.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return helper.Error(c, fiber.StatusUnauthorized, "Pengguna tidak ditemukan")
			}
			return helper.InternalError(c, err)
		}

		// Jika role mahasiswa, pastikan akun mahasiswa tidak dalam status soft delete
		if user.Role == "mahasiswa" {
			student, err := studentRepo.FindByUserID(c.Context(), user.ID)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return helper.Error(c, fiber.StatusUnauthorized, "Akun mahasiswa tidak aktif atau telah dinonaktifkan")
				}
				return helper.InternalError(c, err)
			}
			c.Locals("student", student)
		}

		c.Locals("user", user)
		c.Locals("claims", claims)

		return c.Next()
	}
}

// RequireAdmin memastikan hanya pengguna dengan role admin yang dapat mengakses endpoint.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*helper.JWTClaims)
		if !ok || claims.Role != "admin" {
			return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya admin yang diizinkan")
		}
		return c.Next()
	}
}

// RequireMahasiswa memastikan hanya pengguna dengan role mahasiswa yang dapat mengakses endpoint.
func RequireMahasiswa() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*helper.JWTClaims)
		if !ok || claims.Role != "mahasiswa" {
			return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya mahasiswa yang diizinkan")
		}
		return c.Next()
	}
}
