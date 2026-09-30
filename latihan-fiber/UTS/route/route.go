package route

import (
	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/handler"
	"uts-siakad/app/middleware"
	"uts-siakad/app/repository"
)

type Dependencies struct {
	UserRepo          repository.UserRepository
	StudentRepo       repository.StudentRepository
	AuthHandler       *handler.AuthHandler
	StudentHandler    *handler.StudentHandler
	CourseHandler     *handler.CourseHandler
	EnrollmentHandler *handler.EnrollmentHandler
}

// RegisterRoutes mendaftarkan seluruh 10 endpoint SIAKAD Mini sesuai spesifikasi PDF.
func RegisterRoutes(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// ==========================================
	// 1. Endpoint Autentikasi Publik
	// ==========================================
	// Endpoint 1: POST /api/v1/auth/login (Publik)
	api.Post("/auth/login", deps.AuthHandler.Login)

	// ==========================================
	// Middleware Autentikasi Token untuk Endpoint Terproteksi
	// ==========================================
	authMiddleware := middleware.Authenticate(deps.UserRepo, deps.StudentRepo)
	protected := api.Group("", authMiddleware)

	// Endpoint 2: GET /api/v1/auth/me (Semua role)
	protected.Get("/auth/me", deps.AuthHandler.Me)

	// ==========================================
	// 2. Endpoint Mahasiswa (Students)
	// ==========================================
	// Endpoint 3: GET /api/v1/students (Admin)
	protected.Get("/students", middleware.RequireAdmin(), deps.StudentHandler.List)

	// Endpoint 4: POST /api/v1/students (Admin)
	protected.Post("/students", middleware.RequireAdmin(), deps.StudentHandler.Create)

	// Endpoint 5: GET /api/v1/students/:id (Admin, Mahasiswa data sendiri)
	protected.Get("/students/:id", deps.StudentHandler.Get)

	// Endpoint 6: PUT /api/v1/students/:id (Admin)
	protected.Put("/students/:id", middleware.RequireAdmin(), deps.StudentHandler.Update)

	// Endpoint 7: DELETE /api/v1/students/:id (Admin - Soft delete)
	protected.Delete("/students/:id", middleware.RequireAdmin(), deps.StudentHandler.Delete)

	// ==========================================
	// 3. Endpoint Mata Kuliah (Courses)
	// ==========================================
	// Endpoint 8: GET /api/v1/courses (Semua role)
	protected.Get("/courses", deps.CourseHandler.List)

	// ==========================================
	// 4. Endpoint KRS (Enrollments)
	// ==========================================
	// Endpoint 9: POST /api/v1/enrollments (Mahasiswa)
	protected.Post("/enrollments", middleware.RequireMahasiswa(), deps.EnrollmentHandler.Create)

	// Endpoint 10: DELETE /api/v1/enrollments/:id (Mahasiswa milik sendiri)
	protected.Delete("/enrollments/:id", middleware.RequireMahasiswa(), deps.EnrollmentHandler.Delete)
}
