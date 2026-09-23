package service

import (
	"errors"
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

func terjemahkanUserError(c *fiber.Ctx, err error, fallback string) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	case errors.Is(err, repository.ErrUserExists):
		return helper.Fail(c, fiber.StatusConflict, "username sudah digunakan")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, fallback)
	}
}

func (s *UserService) safeUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}
	return &model.User{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

// ---------- GET /users ----------
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)
	users, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar user")
	}

	safeUsers := make([]model.User, len(users))
	for i, u := range users {
		safeUsers[i] = *s.safeUser(&u)
	}

	totalPages := CountTotalPages(total, q.Limit)

	return helper.OKList(c, "daftar user berhasil diambil", safeUsers, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// ---------- GET /users/:id ----------
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// Pemeriksaan hak akses dilakukan SEBELUM data diambil.
	// Mitigasi timing attack: tidak ada perbedaan waktu tanggap antara id ada dan tidak ada.
	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanUserError(c, err, "gagal mengambil data user")
	}

	return helper.OK(c, "user ditemukan", s.safeUser(user))
}

// ---------- POST /users ----------
func (s *UserService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return helper.Fail(c, fiber.StatusBadRequest, "username dan password wajib diisi")
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "user"
	}
	if !s.perms.IsKnownRole(role) {
		return helper.Fail(c, fiber.StatusBadRequest, "role tidak dikenal")
	}

	pwdHash, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memproses password")
	}

	newUser, err := s.repo.Create(ctx, req.Username, pwdHash, role)
	if err != nil {
		return terjemahkanUserError(c, err, "gagal membuat user")
	}

	return helper.Created(c, "user berhasil dibuat", s.safeUser(newUser), "/api/v1/users/"+string(rune(newUser.ID)))
}

// ---------- PUT /users/:id ----------
func (s *UserService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengubah data user lain")
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return helper.FailValidation(c, map[string]string{"username": "wajib diisi"})
	}

	updated, err := s.repo.Update(ctx, model.User{
		ID:       id,
		Username: req.Username,
	})
	if err != nil {
		return terjemahkanUserError(c, err, "gagal memperbarui user")
	}

	return helper.OK(c, "user berhasil diperbarui", s.safeUser(updated))
}

// ---------- PATCH /users/:id ----------
func (s *UserService) Patch(c *fiber.Ctx) error {
	return s.Replace(c)
}

// ---------- PATCH /users/:id/role ----------
// Dijaga middleware dengan permission role:assign.
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if errs := ValidateAssignRole(current, id, req, s.perms); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	result, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(req.Role))
	if err != nil {
		return terjemahkanUserError(c, err, "gagal mengubah role user")
	}

	return helper.OK(c, "role user berhasil diubah", s.safeUser(result))
}

// ---------- DELETE /users/:id ----------
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// Punya permission menghapus tidak berarti boleh menghapus dirinya sendiri.
	if current.UserID == id {
		return helper.Fail(c, fiber.StatusForbidden, "tidak boleh menghapus akun sendiri")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return terjemahkanUserError(c, err, "gagal menghapus user")
	}

	return helper.NoContent(c)
}
