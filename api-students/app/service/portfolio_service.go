package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
) *PortfolioService {
	return &PortfolioService{
		portfolios: portfolios,
		students:   students,
	}
}

// GET /students/:id/portfolio
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id student harus berupa angka positif")
	}

	// 1. Verifikasi eksistensi student di PostgreSQL
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return translatePortfolioError(err)
	}

	// 2. Ambil portofolio dari MongoDB
	portfolio, err := s.portfolios.GetByStudentID(ctx, studentID)
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.OK(c, "portofolio berhasil diambil", portfolio)
}

// PUT /students/:id/portfolio
func (s *PortfolioService) Upsert(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id student harus berupa angka positif")
	}

	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Headline = strings.TrimSpace(req.Headline)
	req.Bio = strings.TrimSpace(req.Bio)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// 1. Tolak bila mahasiswa tidak ada di PostgreSQL
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return translatePortfolioError(err)
	}

	// 2. Buat atau ganti portofolio di MongoDB
	portfolio, err := s.portfolios.Upsert(ctx, studentID, req.Headline, req.Bio)
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.OK(c, "portofolio berhasil disimpan", portfolio)
}

// POST /students/:id/portfolio/projects
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id student harus berupa angka positif")
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Link = strings.TrimSpace(req.Link)
	req.Technologies = NormalizeTags(req.Technologies)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// 1. Tolak bila mahasiswa tidak ada di PostgreSQL
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		return translatePortfolioError(err)
	}

	// 2. Tambah proyek secara atomik dengan batas 20 proyek
	updated, err := s.portfolios.AddProject(ctx, studentID, model.Project{
		Title:        req.Title,
		Technologies: req.Technologies,
		Link:         req.Link,
		Year:         req.Year,
	})
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.Created(c, "proyek portofolio berhasil ditambahkan", updated, "")
}

// DELETE /students/:id/portfolio/projects/:projectId
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id student harus berupa angka positif")
	}

	projectID := c.Params("projectId")
	if projectID == "" {
		return helper.BadRequest("project id wajib diisi")
	}

	updated, err := s.portfolios.DeleteProject(ctx, studentID, projectID)
	if err != nil {
		return translatePortfolioError(err)
	}

	return helper.OK(c, "proyek portofolio berhasil dihapus", updated)
}

// GET /portfolios/stats/technologies
func (s *PortfolioService) GetStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.GetTopTechnologies(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.OK(c, "statistik teknologi portofolio berhasil diambil", stats)
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data mahasiswa tidak ditemukan di sistem")
	case errors.Is(err, repository.ErrPortfolioNotFound):
		return helper.NotFound("portofolio mahasiswa belum dibuat")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan dalam portofolio")
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("format id proyek tidak valid")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT", "jumlah proyek sudah mencapai batas maksimal 20")
	default:
		return helper.Internal(err)
	}
}
