package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

func main() {
	// 1. Inisialisasi Logger & Konfigurasi Environment
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	config.LoadEnv()

	// 2. Koneksi Basis Data PostgreSQL
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke PostgreSQL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Koneksi Basis Data NoSQL MongoDB (Modul 10)
	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

	// 4. Inisialisasi Repository & Transaction Manager
	txManager := repository.NewTxManager(pool)
	studentRepository := repository.NewStudentRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	roleRepository := repository.NewRoleRepository(pool)
	courseRepository := repository.NewCourseRepository(pool)
	supervisorRepository := repository.NewSupervisorRepository(pool)

	reviewRepository := repository.NewReviewRepository(mongoDB)
	if err := reviewRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index course_reviews MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	portfolioRepository := repository.NewPortfolioRepository(mongoDB)
	if err := portfolioRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index student_portfolios MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5. Memuat Pemetaan Hak Akses (PermissionSet) Sekali Saat Aplikasi Menyala (Fail-Closed)
	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	// 6. Inisialisasi JWT Manager
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET tidak diisi atau terlalu pendek")
	}

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// 7. Inisialisasi Service Layer
	userService := service.NewUserService(userRepository, permissions)
	studentService := service.NewStudentService(studentRepository, permissions)
	authService := service.NewAuthService(
		userRepository,
		tokenRepository,
		jwtManager,
		permissions,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)
	courseService := service.NewCourseService(courseRepository, txManager)
	supervisorService := service.NewSupervisorService(supervisorRepository, studentRepository, txManager)
	reviewService := service.NewReviewService(reviewRepository, courseRepository, permissions)
	portfolioService := service.NewPortfolioService(portfolioRepository, studentRepository)

	// 8. Setup Web Server Fiber & Middleware
	app := fiber.New(fiber.Config{
		AppName:      "API Students, Courses, & MongoDB Reviews/Portfolios (Modul 9 & 10)",
		BodyLimit:    1 * 1024 * 1024,
		ErrorHandler: config.NewErrorHandler(logger),
	})
	app.Use(requestid.New())
	app.Use(middleware.RequestLogger(logger))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// 9. Registrasi Routing Terpusat (Access Control Map)
	route.Register(app, route.Dependencies{
		Pool:              pool,
		Mongo:             mongoClient,
		JWT:               jwtManager,
		Permissions:       permissions,
		UserService:       userService,
		AuthService:       authService,
		StudentService:    studentService,
		CourseService:     courseService,
		SupervisorService: supervisorService,
		ReviewService:     reviewService,
		PortfolioService:  portfolioService,
	})

	// 10. Penangan Route yang Tidak Dikenal (404 Not Found Terpusat)
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	port := config.GetEnv("APP_PORT", "3000")
	logger.Info("Server berjalan", slog.String("port", port))
	log.Fatal(app.Listen(":" + port))
}
