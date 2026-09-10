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

	app := fiber.New(fiber.Config{
		AppName:   "API Students - PostgreSQL & Repository Pattern",
		BodyLimit: 1 * 1024 * 1024,
	})
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

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
	auth := api.Group("/auth")

	auth.Post(
		"/register",
		middleware.RequireJSON,
		authService.Register,
	)

	auth.Post(
		"/login",
		middleware.RequireJSON,
		middleware.LoginRateLimiter(),
		authService.Login,
	)

	auth.Post(
		"/refresh",
		middleware.RequireJSON,
		authService.Refresh,
	)

	auth.Post(
		"/logout",
		middleware.RequireJSON,
		authService.Logout,
	)

	auth.Get(
		"/me",
		middleware.RequireAuth(jwtManager),
		authService.Me,
	)

	students := api.Group(
		"/students",
		middleware.RequireAuth(jwtManager),
	)

	students.Get("/", studentService.List)
	students.Get("/:id", studentService.Get)

	students.Post(
		"/",
		middleware.RequireJSON,
		studentService.Create,
	)

	students.Put(
		"/:id",
		middleware.RequireJSON,
		studentService.Replace,
	)

	students.Patch(
		"/:id",
		middleware.RequireJSON,
		studentService.Patch,
	)

	students.Delete(
		"/:id",
		studentService.Delete,
	)

	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("Server API Students berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}
