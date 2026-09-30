package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"uts-siakad/app/handler"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/repository"
	"uts-siakad/app/service"
	"uts-siakad/config"
	"uts-siakad/database"
	"uts-siakad/route"
)

func main() {
	// 1. Muat konfigurasi
	config.LoadConfig()

	seedFlag := flag.Bool("seed", false, "jalankan migrasi dan seeder data awal")
	flag.Parse()

	// 2. Hubungkan ke PostgreSQL Database Pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Gagal inisialisasi basis data: %v", err)
	}
	defer pool.Close()

	if *seedFlag {
		log.Println("Menjalankan migrasi dan seeder database...")
		if err := database.RunMigrationsAndSeeders(ctx, pool, "."); err != nil {
			log.Fatalf("Gagal menjalankan migrasi & seeder: %v", err)
		}
		log.Println("Seeding selesai.")
	}

	// 3. Inisialisasi Repository Layer
	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	enrollmentRepo := repository.NewEnrollmentRepository(pool)

	// 4. Inisialisasi Service Layer
	authService := service.NewAuthService(userRepo, studentRepo, helper.GlobalLoginLimiter)
	studentService := service.NewStudentService(pool, studentRepo, userRepo, enrollmentRepo)
	courseService := service.NewCourseService(courseRepo)
	enrollmentService := service.NewEnrollmentService(pool, enrollmentRepo, courseRepo, studentRepo)

	// 5. Inisialisasi Handler Layer
	authHandler := handler.NewAuthHandler(authService)
	studentHandler := handler.NewStudentHandler(studentService)
	courseHandler := handler.NewCourseHandler(courseService)
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentService)

	// 6. Konfigurasi Fiber App dengan custom error handler (mencegah kebocoran stack trace di mode production)
	app := fiber.New(fiber.Config{
		AppName: "SIAKAD Mini RESTful API - UTS Backend",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			msg := "Terjadi kesalahan internal pada server"
			if config.AppConfig.AppEnv != "production" {
				msg = err.Error()
			}

			return c.Status(code).JSON(model.SimpleErrorResponse{
				Success: false,
				Message: msg,
				Errors:  nil,
			})
		},
	})

	// Middleware bawaan
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// 7. Registrasi Seluruh 10 Route
	route.RegisterRoutes(app, route.Dependencies{
		UserRepo:          userRepo,
		StudentRepo:       studentRepo,
		AuthHandler:       authHandler,
		StudentHandler:    studentHandler,
		CourseHandler:     courseHandler,
		EnrollmentHandler: enrollmentHandler,
	})

	// 8. Menjalankan Server secara Graceful
	port := config.AppConfig.AppPort
	go func() {
		log.Printf("Server SIAKAD Mini berjalan pada port :%s (Mode: %s)", port, config.AppConfig.AppEnv)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("Server listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Mematikan server dengan graceful shutdown...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Kesalahan saat shutdown server: %v", err)
	}
	fmt.Println("Server berhasil dimatikan dengan aman.")
}
