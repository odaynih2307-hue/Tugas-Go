package handler

import (
	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login menangani POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Format JSON request tidak valid")
	}

	// Validasi input email dan password
	if valErrs := helper.ValidateLoginRequest(req); valErrs != nil {
		return helper.ValidationError(c, valErrs)
	}

	clientIP := c.IP()
	data, statusCode, err := h.authService.Login(c.Context(), req, clientIP)
	if err != nil {
		if statusCode == fiber.StatusTooManyRequests {
			return helper.Error(c, fiber.StatusTooManyRequests, "Terlalu banyak percobaan login yang gagal. Silakan coba lagi nanti.")
		}
		if statusCode == fiber.StatusUnauthorized {
			return helper.Error(c, fiber.StatusUnauthorized, err.Error())
		}
		return helper.InternalError(c, err)
	}

	return helper.Success(c, fiber.StatusOK, "Login berhasil", data)
}

// Me menangani GET /api/v1/auth/me.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*helper.JWTClaims)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Sesi tidak valid")
	}

	data, err := h.authService.Me(c.Context(), claims.UserID)
	if err != nil {
		return helper.InternalError(c, err)
	}

	return helper.Success(c, fiber.StatusOK, "Profil pengguna berhasil diambil", data)
}
