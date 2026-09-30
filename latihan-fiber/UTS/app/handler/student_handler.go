package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/repository"
	"uts-siakad/app/service"
)

type StudentHandler struct {
	studentService service.StudentService
}

func NewStudentHandler(studentService service.StudentService) *StudentHandler {
	return &StudentHandler{studentService: studentService}
}

// List menangani GET /api/v1/students (Admin only).
func (h *StudentHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
	angkatan, _ := strconv.Atoi(c.Query("angkatan", "0"))

	query := model.StudentQuery{
		Page:     page,
		PerPage:  perPage,
		Prodi:    c.Query("prodi", ""),
		Angkatan: angkatan,
		Search:   c.Query("search", ""),
		Sort:     c.Query("sort", ""),
	}

	students, meta, err := h.studentService.List(c.Context(), query)
	if err != nil {
		return helper.InternalError(c, err)
	}

	return helper.SuccessWithMeta(c, fiber.StatusOK, "Data mahasiswa berhasil diambil", students, meta)
}

// Create menangani POST /api/v1/students (Admin only).
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Format JSON request tidak valid")
	}

	if valErrs := helper.ValidateCreateStudentRequest(req); valErrs != nil {
		return helper.ValidationError(c, valErrs)
	}

	student, serviceValErrs, statusCode, err := h.studentService.Create(c.Context(), req)
	if err != nil {
		return helper.InternalError(c, err)
	}
	if serviceValErrs != nil {
		return helper.ValidationError(c, serviceValErrs)
	}

	return helper.Success(c, statusCode, "Mahasiswa dan akun pengguna berhasil dibuat", student)
}

// Get menangani GET /api/v1/students/{id} (Admin, Mahasiswa data sendiri).
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.Error(c, fiber.StatusBadRequest, "ID mahasiswa tidak valid")
	}

	claims, ok := c.Locals("claims").(*helper.JWTClaims)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Sesi tidak valid")
	}

	detail, statusCode, err := h.studentService.GetDetail(c.Context(), id, claims.Role, claims.StudentID)
	if err != nil {
		if statusCode == fiber.StatusForbidden {
			return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda hanya dapat mengakses data mahasiswa milik Anda sendiri")
		}
		if statusCode == fiber.StatusNotFound || errors.Is(err, repository.ErrNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data mahasiswa tidak ditemukan")
		}
		return helper.InternalError(c, err)
	}

	return helper.Success(c, fiber.StatusOK, "Detail data mahasiswa berhasil diambil", detail)
}

// Update menangani PUT /api/v1/students/{id} (Admin only).
func (h *StudentHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.Error(c, fiber.StatusBadRequest, "ID mahasiswa tidak valid")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Format JSON request tidak valid")
	}

	if valErrs := helper.ValidateUpdateStudentRequest(req); valErrs != nil {
		return helper.ValidationError(c, valErrs)
	}

	student, statusCode, err := h.studentService.Update(c.Context(), id, req)
	if err != nil {
		if statusCode == fiber.StatusNotFound || errors.Is(err, repository.ErrNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data mahasiswa tidak ditemukan")
		}
		return helper.InternalError(c, err)
	}

	return helper.Success(c, fiber.StatusOK, "Data mahasiswa berhasil diperbarui", student)
}

// Delete menangani DELETE /api/v1/students/{id} (Admin only - soft delete).
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.Error(c, fiber.StatusBadRequest, "ID mahasiswa tidak valid")
	}

	statusCode, err := h.studentService.SoftDelete(c.Context(), id)
	if err != nil {
		if statusCode == fiber.StatusNotFound || errors.Is(err, repository.ErrNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data mahasiswa tidak ditemukan")
		}
		return helper.InternalError(c, err)
	}

	return helper.NoContent(c)
}
