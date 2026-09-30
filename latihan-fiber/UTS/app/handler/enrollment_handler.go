package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/service"
)

type EnrollmentHandler struct {
	enrollmentService service.EnrollmentService
}

func NewEnrollmentHandler(enrollmentService service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{enrollmentService: enrollmentService}
}

// Create menangani POST /api/v1/enrollments (Mahasiswa only).
func (h *EnrollmentHandler) Create(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*helper.JWTClaims)
	if !ok || claims.Role != "mahasiswa" || claims.StudentID <= 0 {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya akun mahasiswa yang dapat mengambil mata kuliah")
	}

	var req model.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Format JSON request tidak valid")
	}

	if valErrs := helper.ValidateEnrollmentRequest(req); valErrs != nil {
		return helper.ValidationError(c, valErrs)
	}

	enrollment, statusCode, msg, err := h.enrollmentService.Enroll(c.Context(), claims.StudentID, req)
	if err != nil {
		if statusCode == fiber.StatusConflict {
			return helper.Error(c, fiber.StatusConflict, msg)
		}
		if statusCode == fiber.StatusUnprocessableEntity {
			return helper.Error(c, fiber.StatusUnprocessableEntity, msg)
		}
		if statusCode == fiber.StatusNotFound {
			return helper.Error(c, fiber.StatusNotFound, msg)
		}
		return helper.InternalError(c, err)
	}

	return helper.Success(c, statusCode, msg, enrollment)
}

// Delete menangani DELETE /api/v1/enrollments/{id} (Mahasiswa only - milik sendiri).
func (h *EnrollmentHandler) Delete(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*helper.JWTClaims)
	if !ok || claims.Role != "mahasiswa" || claims.StudentID <= 0 {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya akun mahasiswa yang dapat membatalkan mata kuliah")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.Error(c, fiber.StatusBadRequest, "ID enrollment tidak valid")
	}

	statusCode, msg, err := h.enrollmentService.Delete(c.Context(), id, claims.StudentID)
	if err != nil {
		if statusCode == fiber.StatusForbidden {
			return helper.Error(c, fiber.StatusForbidden, msg)
		}
		if statusCode == fiber.StatusNotFound {
			return helper.Error(c, fiber.StatusNotFound, msg)
		}
		return helper.InternalError(c, err)
	}

	return helper.NoContent(c)
}
