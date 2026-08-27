package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"api-students/app/model"
	"api-students/app/repository"
)

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

// terjemahkanError memetakan error repository ke HTTP response status code yang sesuai.
func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return fail(c, fiber.StatusConflict, "NIM sudah digunakan")
	default:
		return fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	q := parseListQuery(c)

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar student")
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return okList(c, "daftar student berhasil diambil", students, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data student")
	}

	return ok(c, "student ditemukan", student)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "NIM wajib diisi"
	}
	if req.Name == "" {
		errs["name"] = "nama wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: isActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal menyimpan student")
	}

	return created(c, "student berhasil dibuat", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "NIM wajib diisi"
	}
	if req.Name == "" {
		errs["name"] = "nama wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	hasil, err := h.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui student")
	}

	return ok(c, "student berhasil diperbarui", hasil)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	if req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	saatIni, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data student")
	}

	if req.NIM != nil {
		trimmed := strings.TrimSpace(*req.NIM)
		if trimmed == "" {
			return failValidation(c, map[string]string{"nim": "NIM tidak boleh kosong"})
		}
		saatIni.NIM = trimmed
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return failValidation(c, map[string]string{"name": "nama tidak boleh kosong"})
		}
		saatIni.Name = trimmed
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return failValidation(c, map[string]string{"grade": "grade harus antara 0 dan 100"})
		}
		saatIni.Grade = *req.Grade
	}

	if req.IsActive != nil {
		saatIni.IsActive = *req.IsActive
	}

	hasil, err := h.repo.Update(ctx, saatIni)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui student")
	}

	return ok(c, "student berhasil diperbarui sebagian", hasil)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "gagal menghapus student")
	}

	return noContent(c)
}
