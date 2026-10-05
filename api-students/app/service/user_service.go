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

type UserService struct {
	repo  *repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserService(
	repo *repository.UserRepository,
	perms *helper.PermissionSet,
) *UserService {
	return &UserService{repo: repo, perms: perms}
}

// translateUserError mengubah error milik repository menjadi AppError terpusat.
//
// Perhatikan tanda tangannya: tidak ada fiber.Ctx. Fungsi ini hanya
// menerjemahkan satu jenis error menjadi jenis lain, dan tidak tahu
// apa pun tentang HTTP. Yang tidak dikenali menjadi Internal — fail
// closed: lebih baik membalas 500 daripada menebak-nebak status.
//
// Catatan Perbaikan Bug Modul Bagian B:
// Pada Bagian B Langkah 4, blok default pada fungsi ini mengembalikan `nil`.
// Akibatnya bila basis data mengalami gangguan tak terduga, handler menganggap
// tidak ada error dan membalas 200 OK dengan body kosong.
// Kami memperbaikinya menjadi: `default: return helper.Internal(err)`.
func translateUserError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound), errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrUserExists), errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("username sudah dipakai")
	default:
		return helper.Internal(err)
	}
}

func (s *UserService) safeUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}
	return &model.User{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}

// ---------- GET /users (Cursor Pagination & Content Negotiation) ----------
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Format dipilih SEBELUM query dijalankan. Bila client meminta format
	// yang tidak dapat kita hasilkan, tidak ada gunanya membebani database
	// untuk hasil yang akan dibuang (Langkah 8).
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	// Baris tambahan hasil limit+1 dipotong di sini. Ia hanya penanda bahwa
	// masih ada halaman berikutnya, bukan bagian dari halaman ini.
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	safeUsers := make([]model.User, len(rows))
	for i, u := range rows {
		safeUsers[i] = *s.safeUser(&u)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(safeUsers) > 0 {
		last := safeUsers[len(safeUsers)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar user berhasil diambil", safeUsers, meta)
}

// ---------- GET /users/:id ----------
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	// Pemeriksaan hak akses dilakukan SEBELUM data diambil.
	// Mitigasi timing attack: tidak ada perbedaan waktu tanggap antara id ada dan tidak ada.
	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user ditemukan", s.safeUser(user))
}

// ---------- POST /users ----------
func (s *UserService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	pwdHash, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	newUser, err := s.repo.Create(ctx, req.Username, pwdHash, "user")
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Created(c, "user berhasil dibuat", s.safeUser(newUser), "/api/v1/users/"+strconv.Itoa(newUser.ID))
}

// ---------- PUT /users/:id ----------
func (s *UserService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengubah data user lain")
	}

	var req model.ReplaceUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.repo.Update(ctx, model.User{
		ID:       id,
		Username: req.Username,
		Email:    req.Email,
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.OK(c, "user berhasil diperbarui", s.safeUser(updated))
}

// ---------- PATCH /users/:id ----------
func (s *UserService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengubah data user lain")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	if IsEmptyUserPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}

	updatedUser := ApplyUserPatch(*saatIni, req)
	updated, err := s.repo.Update(ctx, updatedUser)
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.OK(c, "user berhasil diperbarui sebagian", s.safeUser(updated))
}

// ---------- PATCH /users/:id/role ----------
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	if errs := ValidateAssignRole(current, id, req, s.perms); len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(req.Role))
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.OK(c, "role user berhasil diubah", s.safeUser(result))
}

// ---------- DELETE /users/:id ----------
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	// Punya permission menghapus tidak berarti boleh menghapus dirinya sendiri.
	if current.UserID == id {
		return helper.Forbidden("tidak boleh menghapus akun sendiri")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateUserError(err, "user")
	}

	return helper.NoContent(c)
}
