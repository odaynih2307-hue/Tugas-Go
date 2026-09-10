package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
	"api-students/middleware"
)

func main() {
	// 1. Konfigurasi Environment
	config.LoadEnv()

	// 2. Koneksi Basis Data
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// 3. Dependency Injection
	studentRepository := repository.NewStudentRepository(pool)
	studentService := service.NewStudentService(studentRepository)

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET tidak diisi atau terlalu pendek")
	}

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)

	authService := service.NewAuthService(
		userRepository,
		tokenRepository,
		jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)

	// 4. Inisialisasi Fiber
	app := fiber.New(fiber.Config{
		AppName: "API Students - PostgreSQL & Repository Pattern",
	})
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(cors.New())

	api := app.Group("/api/v1")

	// Endpoint Health Check
	api.Get("/health", func(c *fiber.Ctx) error {

		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(
				c,
				fiber.StatusServiceUnavailable,
				"database tidak dapat dihubungi",
			)
		}

		return helper.OK(c, "server dan database berjalan", nil)
	})
	// Routes Auth
	auth := api.Group("/auth")
	auth.Post("/register", authService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), authService.Login)
	auth.Post("/refresh", authService.Refresh)
	auth.Post("/logout", authService.Logout)
	auth.Get("/me", middleware.RequireAuth(jwtManager), authService.Me)

	// Routes Students
	students := api.Group("/students", middleware.RequireAuth(jwtManager))
	students.Get("/", studentService.List)
	students.Get("/:id", studentService.Get)
	students.Post("/", studentService.Create)
	students.Put("/:id", studentService.Replace)
	students.Patch("/:id", studentService.Patch)
	students.Delete("/:id", studentService.Delete)

	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("Server API Students berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}
