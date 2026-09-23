package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(
	repo repository.StudentRepository,
	perms *helper.PermissionSet,
) *StudentService {
	return &StudentService{
		repo:  repo,
		perms: perms,
	}
}

// terjemahkanError memetakan error repository ke HTTP response yang sesuai.
func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "student tidak ditemukan")

	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah digunakan")

	default:
		return helper.Fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

// ---------- GET /students ----------
// Dijaga oleh middleware RequirePermission(perms, "student:list")
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(
			c,
			fiber.StatusInternalServerError,
			"gagal mengambil daftar student",
		)
	}

	totalPages := CountTotalPages(total, q.Limit)

	return helper.OKList(
		c,
		"daftar student berhasil diambil",
		students,
		&model.Meta{
			Page:       q.Page,
			Limit:      q.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	)
}

// ---------- GET /students/:id ----------
// Memeriksa kepemilikan data: pemilik data selalu boleh, selain itu perlu student:read:any
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"id tidak valid",
		)
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal mengambil data student",
		)
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Fail(
			c,
			fiber.StatusForbidden,
			"tidak berhak mengakses data student ini",
		)
	}

	return helper.OK(c, "student ditemukan", student)
}

// ---------- POST /students ----------
// Dijaga oleh middleware RequirePermission(perms, "student:create")
// owner_id otomatis diisi dari identitas pemanggil (c.Locals / CurrentUser)
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	var req model.CreateStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"format JSON tidak valid",
		)
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// owner_id DIKUNCI dari token autentikasi pemanggil, TIDAK BISA dipalsukan dari request body
	baru, err := s.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: isActive,
		OwnerID:  current.UserID,
	})

	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal menyimpan student",
		)
	}

	return helper.Created(
		c,
		"student berhasil dibuat",
		baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID),
	)
}

// ---------- PUT /students/:id ----------
// Memeriksa kepemilikan data: pemilik data selalu boleh, selain itu perlu student:update:any
func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"id tidak valid",
		)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal mengambil data student",
		)
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(
			c,
			fiber.StatusForbidden,
			"tidak berhak mengubah data student ini",
		)
	}

	var req model.ReplaceStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"format JSON tidak valid",
		)
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})

	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal memperbarui student",
		)
	}

	return helper.OK(
		c,
		"student berhasil diperbarui",
		hasil,
	)
}

// ---------- PATCH /students/:id ----------
// Memeriksa kepemilikan data: pemilik data selalu boleh, selain itu perlu student:update:any
func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"id tidak valid",
		)
	}

	var req model.PatchStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"format JSON tidak valid",
		)
	}

	if IsEmptyPatch(req) {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"tidak ada field yang diubah",
		)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal mengambil data student",
		)
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(
			c,
			fiber.StatusForbidden,
			"tidak berhak mengubah data student ini",
		)
	}

	hasilPatch, errs := ApplyPatch(saatIni, req)

	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hasil, err := s.repo.Update(ctx, hasilPatch)
	if err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal memperbarui student",
		)
	}

	return helper.OK(
		c,
		"student berhasil diperbarui sebagian",
		hasil,
	)
}

// ---------- DELETE /students/:id ----------
// Dijaga oleh middleware RequirePermission(perms, "student:delete")
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(
			c,
			fiber.StatusBadRequest,
			"id tidak valid",
		)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(
			c,
			err,
			"gagal menghapus student",
		)
	}

	return helper.NoContent(c)
}
