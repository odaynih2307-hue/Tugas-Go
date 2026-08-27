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
	"api-students/config"
	"api-students/database"
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

	// 3. Dependency Injection: pool -> repository -> handler
	studentRepository := repository.NewStudentRepository(pool)
	studentHandler := NewStudentHandler(studentRepository)

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
			return fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return ok(c, "server dan database berjalan", nil)
	})

	// Routes Students
	students := api.Group("/students")
	students.Get("/", studentHandler.List)
	students.Get("/:id", studentHandler.Get)
	students.Post("/", studentHandler.Create)
	students.Put("/:id", studentHandler.Replace)
	students.Patch("/:id", studentHandler.Patch)
	students.Delete("/:id", studentHandler.Delete)

	port := config.GetEnv("APP_PORT", "3000")
	log.Printf("Server API Students berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}
