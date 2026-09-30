package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
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

var (
	testApp     *fiber.App
	adminToken  string
	mhsToken    string
	mhs2Token   string
	mhsID       int
	mhs2ID      int
)

func TestMain(m *testing.M) {
	// Set environment variables for testing
	os.Setenv("APP_ENV", "testing")
	os.Setenv("DB_NAME", "siakad_mini")
	config.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Printf("Gagal koneksi ke database pengujian: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Pastikan skema dan data awal bersih & ter-seed
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE enrollments, students, users RESTART IDENTITY CASCADE;")
	_ = database.RunMigrationsAndSeeders(ctx, pool, "..")

	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	enrollmentRepo := repository.NewEnrollmentRepository(pool)

	authService := service.NewAuthService(userRepo, studentRepo, helper.GlobalLoginLimiter)
	studentService := service.NewStudentService(pool, studentRepo, userRepo, enrollmentRepo)
	courseService := service.NewCourseService(courseRepo)
	enrollmentService := service.NewEnrollmentService(pool, enrollmentRepo, courseRepo, studentRepo)

	authHandler := handler.NewAuthHandler(authService)
	studentHandler := handler.NewStudentHandler(studentService)
	courseHandler := handler.NewCourseHandler(courseService)
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentService)

	app := fiber.New(fiber.Config{
		AppName: "UTS Testing Suite",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(model.SimpleErrorResponse{
				Success: false,
				Message: err.Error(),
				Errors:  nil,
			})
		},
	})
	app.Use(recover.New())

	route.RegisterRoutes(app, route.Dependencies{
		UserRepo:          userRepo,
		StudentRepo:       studentRepo,
		AuthHandler:       authHandler,
		StudentHandler:    studentHandler,
		CourseHandler:     courseHandler,
		EnrollmentHandler: enrollmentHandler,
	})

	testApp = app

	// Ambil ID mahasiswa Rina Putri (NIM 187221000001) dan Budi Santoso (NIM 187221000002)
	s1, _, err := studentRepo.FindAll(ctx, model.StudentQuery{Search: "187221000001", Page: 1, PerPage: 1})
	if err == nil && len(s1) > 0 {
		mhsID = s1[0].ID
	}
	s2, _, err := studentRepo.FindAll(ctx, model.StudentQuery{Search: "187221000002", Page: 1, PerPage: 1})
	if err == nil && len(s2) > 0 {
		mhs2ID = s2[0].ID
	}

	code := m.Run()
	os.Exit(code)
}

func sendRequest(app *fiber.App, method, url, token string, body any) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req, 10000)
	if err != nil {
		return nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	return resp, respBody, err
}

// =========================================================================
// Test 1: POST /api/v1/auth/login
// =========================================================================
func TestEndpoint1_Login(t *testing.T) {
	t.Run("1.1 Admin Login Sukses (200)", func(t *testing.T) {
		loginReq := model.LoginRequest{
			Email:    "admin@siakad.ac.id",
			Password: "admin12345",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Success bool                    `json:"success"`
			Data    model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(body, &res)
		if !res.Success || res.Data.AccessToken == "" || res.Data.User.Role != "admin" {
			t.Errorf("Invalid login response: %s", string(body))
		}
		adminToken = res.Data.AccessToken
	})

	t.Run("1.2 Mahasiswa Login Sukses (200)", func(t *testing.T) {
		loginReq := model.LoginRequest{
			Email:    "rina.putri@siakad.ac.id",
			Password: "187221000001",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Success bool                    `json:"success"`
			Data    model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(body, &res)
		if !res.Success || res.Data.AccessToken == "" || res.Data.User.Role != "mahasiswa" {
			t.Errorf("Invalid mahasiswa login response: %s", string(body))
		}
		mhsToken = res.Data.AccessToken

		// Login mahasiswa 2 juga untuk pengujian otorisasi
		loginReq2 := model.LoginRequest{
			Email:    "budi.santoso@siakad.ac.id",
			Password: "187221000002",
		}
		_, body2, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq2)
		var res2 struct {
			Data model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(body2, &res2)
		mhs2Token = res2.Data.AccessToken
	})

	t.Run("1.3 Login Gagal - Kredensial Salah (401)", func(t *testing.T) {
		loginReq := model.LoginRequest{
			Email:    "admin@siakad.ac.id",
			Password: "wrongpassword123",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("1.4 Login Gagal - Validasi Email & Password (422)", func(t *testing.T) {
		loginReq := model.LoginRequest{
			Email:    "bukan-email",
			Password: "123", // kurang dari 8 karakter
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("1.5 Rate Limiting Login Gagal > 5 Kali (429)", func(t *testing.T) {
		failEmail := "rate.test@siakad.ac.id"
		req := model.LoginRequest{Email: failEmail, Password: "wrongpassword123"}

		var lastStatus int
		for i := 0; i < 6; i++ {
			resp, _, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", req)
			lastStatus = resp.StatusCode
		}

		if lastStatus != http.StatusTooManyRequests {
			t.Errorf("Expected 429 after >5 failed attempts, got %d", lastStatus)
		}
	})
}

// =========================================================================
// Test 2: GET /api/v1/auth/me
// =========================================================================
func TestEndpoint2_AuthMe(t *testing.T) {
	t.Run("2.1 Admin Mengakses /me (200)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/auth/me", adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("2.2 Mahasiswa Mengakses /me beserta Profil Mahasiswa (200)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/auth/me", mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data model.AuthMeResponse `json:"data"`
		}
		json.Unmarshal(body, &res)
		if res.Data.Student == nil || res.Data.Student.NIM != "187221000001" {
			t.Errorf("Expected student profile data attached, got: %s", string(body))
		}
	})

	t.Run("2.3 Tanpa Token Otorisasi (401)", func(t *testing.T) {
		resp, _, err := sendRequest(testApp, "GET", "/api/v1/auth/me", "", nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 3: GET /api/v1/students
// =========================================================================
func TestEndpoint3_GetStudents(t *testing.T) {
	t.Run("3.1 Admin Melihat Daftar Mahasiswa dengan Meta Pagination (200)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/students?page=1&per_page=10", adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data []model.StudentListItem `json:"data"`
			Meta model.Meta              `json:"meta"`
		}
		json.Unmarshal(body, &res)
		if len(res.Data) == 0 || res.Meta.Total < 20 {
			t.Errorf("Expected >=20 students, got %d total in meta", res.Meta.Total)
		}
	})

	t.Run("3.2 Filter Search dan Sorting Mahasiswa (200)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/students?search=Rina&sort=-ipk_terakhir", adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("3.3 Akses Ditolak untuk Mahasiswa (403)", func(t *testing.T) {
		resp, _, err := sendRequest(testApp, "GET", "/api/v1/students", mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 4: POST /api/v1/students
// =========================================================================
var createdStudentID int

func TestEndpoint4_CreateStudent(t *testing.T) {
	t.Run("4.1 Admin Menambah Mahasiswa Baru (201)", func(t *testing.T) {
		ipk := 3.75
		newStudent := model.CreateStudentRequest{
			NIM:         "187221009999",
			Nama:        "Testing Mahasiswa Baru",
			Email:       "test.mahasiswa99@siakad.ac.id",
			Prodi:       "Teknik Informatika",
			Angkatan:    2023,
			IPKTerakhir: &ipk,
		}

		resp, body, err := sendRequest(testApp, "POST", "/api/v1/students", adminToken, newStudent)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data model.Student `json:"data"`
		}
		json.Unmarshal(body, &res)
		createdStudentID = res.Data.ID

		// Uji login dengan user baru (password default = NIM)
		loginReq := model.LoginRequest{
			Email:    newStudent.Email,
			Password: newStudent.NIM,
		}
		loginResp, _, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if loginResp.StatusCode != http.StatusOK {
			t.Errorf("New student user should be able to login with NIM as password, got: %d", loginResp.StatusCode)
		}
	})

	t.Run("4.2 Validasi Duplikasi NIM dan Email (422)", func(t *testing.T) {
		ipk := 3.50
		dupStudent := model.CreateStudentRequest{
			NIM:         "187221009999", // duplikat
			Nama:        "Duplikat NIM",
			Email:       "test.mahasiswa99@siakad.ac.id", // duplikat
			Prodi:       "Sistem Informasi",
			Angkatan:    2023,
			IPKTerakhir: &ipk,
		}

		resp, body, err := sendRequest(testApp, "POST", "/api/v1/students", adminToken, dupStudent)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("4.3 Mahasiswa Mencoba Menambah Mahasiswa (403)", func(t *testing.T) {
		ipk := 3.50
		req := model.CreateStudentRequest{
			NIM:         "187221008888",
			Nama:        "Unauthorized Mhs",
			Email:       "unauth@siakad.ac.id",
			Prodi:       "Sistem Informasi",
			Angkatan:    2023,
			IPKTerakhir: &ipk,
		}
		resp, _, err := sendRequest(testApp, "POST", "/api/v1/students", mhsToken, req)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 5: GET /api/v1/students/{id}
// =========================================================================
func TestEndpoint5_GetStudentDetail(t *testing.T) {
	t.Run("5.1 Mahasiswa Mengakses Data Milik Sendiri beserta Total & Batas SKS (200)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/students/%d", mhsID)
		resp, body, err := sendRequest(testApp, "GET", url, mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data model.StudentDetailResponse `json:"data"`
		}
		json.Unmarshal(body, &res)
		if res.Data.BatasSKS != 24 { // Rina Putri IPK 3.45 -> Batas SKS 24
			t.Errorf("Expected BatasSKS 24 for IPK 3.45, got %d", res.Data.BatasSKS)
		}
	})

	t.Run("5.2 Mahasiswa Mengakses Data Mahasiswa Lain (403)", func(t *testing.T) {
		// mhsToken (Rina) mencoba mengakses profil mhs2ID (Budi)
		url := fmt.Sprintf("/api/v1/students/%d", mhs2ID)
		resp, body, err := sendRequest(testApp, "GET", url, mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("5.3 Admin Mengakses Data Mahasiswa Siapa Saja (200)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/students/%d", mhsID)
		resp, _, err := sendRequest(testApp, "GET", url, adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 6: PUT /api/v1/students/{id}
// =========================================================================
func TestEndpoint6_UpdateStudent(t *testing.T) {
	t.Run("6.1 Admin Memperbarui Data Mahasiswa (200)", func(t *testing.T) {
		newIPK := 3.95
		updateReq := model.UpdateStudentRequest{
			Nama:        "Testing Mahasiswa Updated",
			Prodi:       "Sains Data",
			Angkatan:    2023,
			IPKTerakhir: &newIPK,
		}

		url := fmt.Sprintf("/api/v1/students/%d", createdStudentID)
		resp, body, err := sendRequest(testApp, "PUT", url, adminToken, updateReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("6.2 Mahasiswa Mencoba Memperbarui Data (403)", func(t *testing.T) {
		updateReq := model.UpdateStudentRequest{
			Nama:     "Hacker Update",
			Prodi:    "Sains Data",
			Angkatan: 2023,
		}
		url := fmt.Sprintf("/api/v1/students/%d", mhsID)
		resp, _, err := sendRequest(testApp, "PUT", url, mhsToken, updateReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 7: DELETE /api/v1/students/{id}
// =========================================================================
func TestEndpoint7_SoftDeleteStudent(t *testing.T) {
	t.Run("7.1 Admin Melakukan Soft Delete Mahasiswa (204)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/students/%d", createdStudentID)
		resp, _, err := sendRequest(testApp, "DELETE", url, adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("7.2 Mahasiswa yang Telah Di-Soft Delete Tidak Muncul di List Mahasiswa", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/students?search=187221009999", adminToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
		var res struct {
			Data []model.StudentListItem `json:"data"`
		}
		json.Unmarshal(body, &res)
		if len(res.Data) > 0 {
			t.Errorf("Soft-deleted student should not appear in student list")
		}
	})

	t.Run("7.3 Mahasiswa yang Telah Di-Soft Delete Tidak Dapat Login (401)", func(t *testing.T) {
		loginReq := model.LoginRequest{
			Email:    "test.mahasiswa99@siakad.ac.id",
			Password: "187221009999",
		}
		resp, _, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginReq)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Soft-deleted student should not be able to login, expected 401, got %d", resp.StatusCode)
		}
	})
}

// =========================================================================
// Test 8: GET /api/v1/courses
// =========================================================================
func TestEndpoint8_GetCourses(t *testing.T) {
	t.Run("8.1 Mengambil Daftar Mata Kuliah beserta Terisi dan Sisa Kuota (200)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/courses", mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data []model.CourseResponse `json:"data"`
		}
		json.Unmarshal(body, &res)
		if len(res.Data) < 10 {
			t.Errorf("Expected at least 10 courses, got %d", len(res.Data))
		}

		// Cek kalkulasi sisa kuota = kuota - terisi
		for _, c := range res.Data {
			if c.SisaKuota != c.Kuota-c.Terisi {
				t.Errorf("Course %s calculation mismatch: kuota=%d, terisi=%d, sisa=%d", c.KodeMK, c.Kuota, c.Terisi, c.SisaKuota)
			}
		}
	})

	t.Run("8.2 Filter Mata Kuliah Tersedia (available=true)", func(t *testing.T) {
		resp, body, err := sendRequest(testApp, "GET", "/api/v1/courses?available=true", mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
}

// =========================================================================
// Test 9 & 10: POST /api/v1/enrollments & DELETE /api/v1/enrollments/{id}
// =========================================================================
var testEnrollmentID int

func TestEndpoint9_and_10_Enrollments(t *testing.T) {
	// Course ID 1: Algoritma dan Pemrograman (3 SKS)
	// Course ID 8: IF402 Kuota 2
	t.Run("9.1 Mahasiswa Mengambil Mata Kuliah ke KRS (201)", func(t *testing.T) {
		req := model.CreateEnrollmentRequest{
			CourseID:      1,
			TahunAkademik: "2026/2027-Ganjil",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/enrollments", mhsToken, req)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Data model.Enrollment `json:"data"`
		}
		json.Unmarshal(body, &res)
		testEnrollmentID = res.Data.ID
	})

	t.Run("9.2 Menolak Pengambilan Mata Kuliah Duplikat pada Tahun Akademik Sama (409)", func(t *testing.T) {
		req := model.CreateEnrollmentRequest{
			CourseID:      1, // sudah diambil di 9.1
			TahunAkademik: "2026/2027-Ganjil",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/enrollments", mhsToken, req)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected 409, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("9.3 Menolak Pengambilan oleh Role Admin (403)", func(t *testing.T) {
		req := model.CreateEnrollmentRequest{
			CourseID:      2,
			TahunAkademik: "2026/2027-Ganjil",
		}
		resp, _, err := sendRequest(testApp, "POST", "/api/v1/enrollments", adminToken, req)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("9.4 Validasi Batas SKS Terlampaui (422 dengan pesan sisa SKS)", func(t *testing.T) {
		// Tiara Andini (NIM 187221000017) memiliki IPK 1.95 (< 2.50) -> Batas SKS = 18 SKS
		// Login sebagai Tiara
		loginTiara := model.LoginRequest{Email: "tiara.andini@siakad.ac.id", Password: "187221000017"}
		_, loginBody, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginTiara)
		var tiaraRes struct {
			Data model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(loginBody, &tiaraRes)

		// Ambil beberapa mata kuliah sampai mendekati 18 SKS
		// IF502 = 6 SKS, IF102 = 4 SKS, IF402 = 4 SKS, IF101 = 3 SKS (total 17 SKS)
		sendRequest(testApp, "POST", "/api/v1/enrollments", tiaraRes.Data.AccessToken, model.CreateEnrollmentRequest{CourseID: 10, TahunAkademik: "2026/2027-Ganjil"}) // 6 SKS
		sendRequest(testApp, "POST", "/api/v1/enrollments", tiaraRes.Data.AccessToken, model.CreateEnrollmentRequest{CourseID: 2, TahunAkademik: "2026/2027-Ganjil"})  // 4 SKS
		sendRequest(testApp, "POST", "/api/v1/enrollments", tiaraRes.Data.AccessToken, model.CreateEnrollmentRequest{CourseID: 8, TahunAkademik: "2026/2027-Ganjil"})  // 4 SKS
		sendRequest(testApp, "POST", "/api/v1/enrollments", tiaraRes.Data.AccessToken, model.CreateEnrollmentRequest{CourseID: 1, TahunAkademik: "2026/2027-Ganjil"})  // 3 SKS -> Total: 17 SKS, Sisa: 1 SKS

		// Mencoba mengambil IF301 (3 SKS) -> 17 + 3 = 20 > 18 (melebihi batas SKS)
		overReq := model.CreateEnrollmentRequest{
			CourseID:      5, // 3 SKS
			TahunAkademik: "2026/2027-Ganjil",
		}
		resp, body, err := sendRequest(testApp, "POST", "/api/v1/enrollments", tiaraRes.Data.AccessToken, overReq)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for SKS limit exceeded, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("9.5 Validasi Kuota Mata Kuliah Penuh (422)", func(t *testing.T) {
		// IF402 memiliki kuota = 2.
		// Ambil akun mahasiswa lain untuk memenuhi kuota
		// Budi Santoso & Siti Nurhaliza mengambil IF402
		reqCourse8 := model.CreateEnrollmentRequest{
			CourseID:      8, // Kuota: 2
			TahunAkademik: "2026/2027-Ganjil",
		}
		// Budi (mhs2Token) ambil
		sendRequest(testApp, "POST", "/api/v1/enrollments", mhs2Token, reqCourse8)

		// Dewi Lestari (NIM 187221000005) ambil kursi ke-2
		loginDewi := model.LoginRequest{Email: "dewi.lestari@siakad.ac.id", Password: "187221000005"}
		_, dewiBody, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginDewi)
		var dewiRes struct {
			Data model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(dewiBody, &dewiRes)
		sendRequest(testApp, "POST", "/api/v1/enrollments", dewiRes.Data.AccessToken, reqCourse8)

		// Sekarang kuota IF402 sudah terisi 2 (penuh).
		// Rizky Pratama (NIM 187221000006) mencoba mengambil -> Harus ditolak 422 Kuota Penuh
		loginRizky := model.LoginRequest{Email: "rizky.pratama@siakad.ac.id", Password: "187221000006"}
		_, rizkyBody, _ := sendRequest(testApp, "POST", "/api/v1/auth/login", "", loginRizky)
		var rizkyRes struct {
			Data model.LoginResponseData `json:"data"`
		}
		json.Unmarshal(rizkyBody, &rizkyRes)

		resp, body, err := sendRequest(testApp, "POST", "/api/v1/enrollments", rizkyRes.Data.AccessToken, reqCourse8)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 when quota is full, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("10.1 Mahasiswa Membatalkan KRS Milik Mahasiswa Lain (403)", func(t *testing.T) {
		// mhs2Token (Budi) mencoba menghapus enrollment milik mhsToken (Rina)
		url := fmt.Sprintf("/api/v1/enrollments/%d", testEnrollmentID)
		resp, body, err := sendRequest(testApp, "DELETE", url, mhs2Token, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("10.2 Mahasiswa Membatalkan Mata Kuliah dari KRS Sendiri (204)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/enrollments/%d", testEnrollmentID)
		resp, _, err := sendRequest(testApp, "DELETE", url, mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("10.3 Membatalkan Enrollment yang Tidak Ditemukan (404)", func(t *testing.T) {
		resp, _, err := sendRequest(testApp, "DELETE", "/api/v1/enrollments/99999", mhsToken, nil)
		if err != nil {
			t.Fatalf("Request error: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", resp.StatusCode)
		}
	})
}
