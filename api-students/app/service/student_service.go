package service

import (
	"errors"
	"strconv"

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

// translateStudentError memetakan error repository ke AppError terpusat.
func translateStudentError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah digunakan")
	default:
		return helper.Internal(err)
	}
}

// ---------- GET /students (Keyset Cursor Pagination & Content Negotiation) ----------
// Dijaga oleh middleware RequirePermission(perms, "student:list")
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// 1. Content Negotiation (Langkah 8 & Tugas D.4)
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	// 2. Keyset Cursor Pagination (Langkah 7 & Tugas D.3)
	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	// 3. Ekspor format CSV jika diminta Accept: text/csv
	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, rows)
	}

	// 4. Potong baris tambahan limit+1 untuk menghitung has_more & next_cursor
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar student berhasil diambil", rows, meta)
}

// ---------- GET /students/:id ----------
// Memeriksa kepemilikan data: pemilik data selalu boleh, selain itu perlu student:read:any
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "student")
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Forbidden("tidak berhak mengakses data student ini")
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
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	// Validasi deklaratif menggunakan tag struct (Tugas D.2)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
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
		return translateStudentError(err, "student")
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
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "student")
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data student ini")
	}

	var req model.ReplaceStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	// Validasi deklaratif menggunakan tag struct (Tugas D.2)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})

	if err != nil {
		return translateStudentError(err, "student")
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
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	var req model.PatchStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	if IsEmptyPatchStudent(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	// Validasi deklaratif dengan tag struct dan omitnil (Tugas D.2 butir 3)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "student")
	}

	// Pemeriksaan Kepemilikan (Ownership) & Permission
	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data student ini")
	}

	hasilPatch := ApplyPatchStudent(saatIni, req)

	hasil, err := s.repo.Update(ctx, hasilPatch)
	if err != nil {
		return translateStudentError(err, "student")
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
		return helper.BadRequest("id tidak valid")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateStudentError(err, "student")
	}

	return helper.NoContent(c)
}
