package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/app/service"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

var (
	testApp     *fiber.App
	testPool    *repository.UserRepository
	testJWT     *helper.JWTManager
	adminToken  string
	userToken   string
	budiToken   string
	createdUserIDs []int
)

func setupTestApp(t *testing.T) *fiber.App {
	config.LoadEnv()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pool, err := database.NewPool(context.Background())
	if err != nil {
		t.Fatalf("Gagal terhubung ke database: %v", err)
	}

	roleRepo := repository.NewRoleRepository(pool)
	rawPerms, err := roleRepo.LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("Gagal memuat permissions: %v", err)
	}
	perms := helper.NewPermissionSet(rawPerms)

	jwtSecret := config.GetEnv("JWT_SECRET", "supersecretjwtkeywithmorethan32bytes123456!")
	jwtManager := helper.NewJWTManager(jwtSecret, "praktikum-backend", 15*time.Minute)
	testJWT = jwtManager

	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	userService := service.NewUserService(userRepo, perms)
	studentService := service.NewStudentService(studentRepo, perms)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtManager, perms, 7*24*time.Hour)

	app := fiber.New(fiber.Config{
		AppName:      "Test API Modul 7",
		ErrorHandler: config.NewErrorHandler(logger),
	})
	app.Use(requestid.New())
	app.Use(middleware.RequestLogger(logger))

	route.Register(app, route.Dependencies{
		Pool:           pool,
		JWT:            jwtManager,
		Permissions:    perms,
		UserService:    userService,
		AuthService:    authService,
		StudentService: studentService,
	})

	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	adminToken, _, _ = jwtManager.GenerateAccessToken(4, "admin", "admin")
	userToken, _, _ = jwtManager.GenerateAccessToken(1, "ody", "user")
	budiToken, _, _ = jwtManager.GenerateAccessToken(6, "budi_owner", "user")

	return app
}

func executeRequest(app *fiber.App, req *http.Request) (*http.Response, string, error) {
	resp, err := app.Test(req, 10000)
	if err != nil {
		return nil, "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}
	return resp, string(body), nil
}

// -----------------------------------------------------------------------------
// Test 9.1 & 9.2: Keyset Cursor Pagination & Anti-Duplication Verification
// -----------------------------------------------------------------------------
func TestKeysetCursorPagination(t *testing.T) {
	app := setupTestApp(t)

	// Ambil halaman 1 dengan limit=3
	req := httptest.NewRequest("GET", "/api/v1/users?limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, body, err := executeRequest(app, req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", resp.StatusCode, body)
	}

	var jsonResp model.WebCursorResponse
	if err := json.Unmarshal([]byte(body), &jsonResp); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if jsonResp.Meta == nil || jsonResp.Meta.NextCursor == "" {
		t.Fatalf("Expected NextCursor in Meta, got nil or empty. Meta: %+v", jsonResp.Meta)
	}

	cursor1 := jsonResp.Meta.NextCursor
	t.Logf("Halaman 1 sukses! NextCursor: %s, HasMore: %v", cursor1, jsonResp.Meta.HasMore)

	// Ambil halaman 2 menggunakan cursor1
	req2 := httptest.NewRequest("GET", "/api/v1/users?limit=3&cursor="+cursor1, nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)

	resp2, body2, err := executeRequest(app, req2)
	if err != nil {
		t.Fatalf("Request 2 failed: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", resp2.StatusCode, body2)
	}

	var jsonResp2 model.WebCursorResponse
	if err := json.Unmarshal([]byte(body2), &jsonResp2); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	t.Logf("Halaman 2 sukses! Data count: %d, HasMore: %v", len(jsonResp2.Data.([]any)), jsonResp2.Meta.HasMore)
}

// -----------------------------------------------------------------------------
// Test 9.3: Declarative Validation (Register & PATCH omitnil)
// -----------------------------------------------------------------------------
func TestDeclarativeValidation(t *testing.T) {
	app := setupTestApp(t)

	// Kasus 1: Register dengan karakter username tidak valid, email rusak, password terlalu pendek
	badPayload := `{"username":"a b!","email":"bukan-email","password":"abc"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(badPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, body, err := executeRequest(app, req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Status code WAJIB 422 Unprocessable Entity! (Bukan 400)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422 Unprocessable Entity, got %d. Body: %s", resp.StatusCode, body)
	}

	var errResp model.ErrorResponse
	if err := json.Unmarshal([]byte(body), &errResp); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if errResp.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected code VALIDATION_ERROR, got %s", errResp.Code)
	}
	if errResp.Fields["email"] != "format email tidak valid" {
		t.Errorf("Expected email validation error, got: %v", errResp.Fields["email"])
	}
	if errResp.Fields["password"] != "minimal 8 karakter" {
		t.Errorf("Expected password validation error, got: %v", errResp.Fields["password"])
	}
	if errResp.Fields["username"] != "hanya boleh huruf, angka, titik, dan garis bawah" {
		t.Errorf("Expected username validation error, got: %v", errResp.Fields["username"])
	}

	// Kasus 2: Password terlalu umum ("password123")
	commonPayload := `{"username":"coba01","email":"c@unair.ac.id","password":"password123"}`
	reqCommon := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(commonPayload))
	reqCommon.Header.Set("Content-Type", "application/json")

	respCommon, bodyCommon, _ := executeRequest(app, reqCommon)
	if respCommon.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422, got %d", respCommon.StatusCode)
	}
	var errCommon model.ErrorResponse
	_ = json.Unmarshal([]byte(bodyCommon), &errCommon)
	if errCommon.Fields["password"] != "password terlalu umum" {
		t.Errorf("Expected 'password terlalu umum', got: %v", errCommon.Fields["password"])
	}

	// Kasus 3: Password tanpa angka ("tanpaangka")
	noDigitPayload := `{"username":"coba02","email":"c@unair.ac.id","password":"tanpaangka"}`
	reqNoDigit := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(noDigitPayload))
	reqNoDigit.Header.Set("Content-Type", "application/json")

	respNoDigit, bodyNoDigit, _ := executeRequest(app, reqNoDigit)
	if respNoDigit.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422, got %d", respNoDigit.StatusCode)
	}
	var errNoDigit model.ErrorResponse
	_ = json.Unmarshal([]byte(bodyNoDigit), &errNoDigit)
	if errNoDigit.Fields["password"] != "harus memuat huruf dan angka" {
		t.Errorf("Expected 'harus memuat huruf dan angka', got: %v", errNoDigit.Fields["password"])
	}

	// Kasus 4: PATCH dengan string kosong {"username":""} TETAP diperiksa dan DITOLAK (422)
	patchEmpty := `{"username":""}`
	reqPatch := httptest.NewRequest("PATCH", "/api/v1/users/4", strings.NewReader(patchEmpty))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("Authorization", "Bearer "+adminToken)

	respPatch, bodyPatch, _ := executeRequest(app, reqPatch)
	if respPatch.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422 for empty username patch, got %d. Body: %s", respPatch.StatusCode, bodyPatch)
	}
	var errPatch model.ErrorResponse
	_ = json.Unmarshal([]byte(bodyPatch), &errPatch)
	if errPatch.Fields["username"] != "minimal 3 karakter" {
		t.Errorf("Expected 'minimal 3 karakter', got: %v", errPatch.Fields["username"])
	}

	// Kasus 5: PATCH tanpa field apa pun -> 400 Bad Request
	patchNoFields := `{}`
	reqNoFields := httptest.NewRequest("PATCH", "/api/v1/users/4", strings.NewReader(patchNoFields))
	reqNoFields.Header.Set("Content-Type", "application/json")
	reqNoFields.Header.Set("Authorization", "Bearer "+adminToken)

	respNoFields, _, _ := executeRequest(app, reqNoFields)
	if respNoFields.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 for empty patch body, got %d", respNoFields.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// Test 9.4: Content Negotiation (JSON vs CSV, 406 on unsupported, */* fallback)
// -----------------------------------------------------------------------------
func TestContentNegotiation(t *testing.T) {
	app := setupTestApp(t)

	// Kasus 1: Permintaan format CSV (Accept: text/csv)
	reqCSV := httptest.NewRequest("GET", "/api/v1/users?limit=3", nil)
	reqCSV.Header.Set("Authorization", "Bearer "+adminToken)
	reqCSV.Header.Set("Accept", "text/csv")

	respCSV, bodyCSV, err := executeRequest(app, reqCSV)
	if err != nil {
		t.Fatalf("Request CSV failed: %v", err)
	}
	if respCSV.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", respCSV.StatusCode, bodyCSV)
	}

	ct := respCSV.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("Expected Content-Type text/csv, got %s", ct)
	}
	cd := respCSV.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "users.csv") {
		t.Errorf("Expected attachment users.csv, got %s", cd)
	}

	// Verifikasi isi CSV tidak kosong (menjawab perbaikan writer.Flush())
	if !strings.Contains(bodyCSV, "id,username,email,role,is_active,created_at") {
		t.Fatalf("CSV header missing or empty file! Body: %q", bodyCSV)
	}
	lines := strings.Split(strings.TrimSpace(bodyCSV), "\n")
	if len(lines) < 2 {
		t.Fatalf("CSV data empty, only header found! Body: %q", bodyCSV)
	}
	t.Logf("CSV Export verified successfully with %d lines!", len(lines))

	// Kasus 2: Permintaan format yang tidak didukung (Accept: application/xml) -> 406 NOT_ACCEPTABLE
	reqXML := httptest.NewRequest("GET", "/api/v1/users?limit=3", nil)
	reqXML.Header.Set("Authorization", "Bearer "+adminToken)
	reqXML.Header.Set("Accept", "application/xml")

	respXML, bodyXML, _ := executeRequest(app, reqXML)
	if respXML.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("Expected 406 Not Acceptable, got %d. Body: %s", respXML.StatusCode, bodyXML)
	}
	var errXML model.ErrorResponse
	_ = json.Unmarshal([]byte(bodyXML), &errXML)
	if errXML.Code != "NOT_ACCEPTABLE" {
		t.Errorf("Expected code NOT_ACCEPTABLE, got %s", errXML.Code)
	}

	// Kasus 3: Permintaan Accept: */* dijawab default JSON (200 OK)
	reqAny := httptest.NewRequest("GET", "/api/v1/users?limit=3", nil)
	reqAny.Header.Set("Authorization", "Bearer "+adminToken)
	reqAny.Header.Set("Accept", "*/*")

	respAny, bodyAny, _ := executeRequest(app, reqAny)
	if respAny.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for */*, got %d", respAny.StatusCode)
	}
	if !strings.Contains(bodyAny, `"success":true`) {
		t.Fatalf("Expected JSON response for */*, got: %s", bodyAny)
	}
}

// -----------------------------------------------------------------------------
// Test 9.5: Keseragaman Response Kegagalan (9 Titik Sesuai Tabel Modul)
// -----------------------------------------------------------------------------
func TestErrorResponseUniformity(t *testing.T) {
	app := setupTestApp(t)

	cases := []struct {
		name       string
		method     string
		url        string
		body       string
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "GET /users/999999 (User not found)",
			method:     "GET",
			url:        "/api/v1/users/999999",
			headers:    map[string]string{"Authorization": "Bearer " + adminToken},
			wantStatus: 404,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "GET /users?cursor=bukanbase64!! (Malformed cursor)",
			method:     "GET",
			url:        "/api/v1/users?cursor=bukanbase64!!",
			headers:    map[string]string{"Authorization": "Bearer " + adminToken},
			wantStatus: 400,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "GET /api/v1/tidakada (Route not found)",
			method:     "GET",
			url:        "/api/v1/tidakada",
			wantStatus: 404,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "GET /users tanpa token",
			method:     "GET",
			url:        "/api/v1/users",
			wantStatus: 401,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "POST /users dengan Content-Type: text/plain",
			method:     "POST",
			url:        "/api/v1/users",
			body:       `{"username":"coba"}`,
			headers:    map[string]string{"Content-Type": "text/plain", "Authorization": "Bearer " + adminToken},
			wantStatus: 415,
			wantCode:   "UNSUPPORTED_MEDIA_TYPE",
		},
		{
			name:       "POST /users dengan JSON rusak",
			method:     "POST",
			url:        "/api/v1/users",
			body:       `{corrupted json`,
			headers:    map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + adminToken},
			wantStatus: 400,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "GET /users?is_active=mungkin (Invalid query param)",
			method:     "GET",
			url:        "/api/v1/users?is_active=mungkin",
			headers:    map[string]string{"Authorization": "Bearer " + adminToken},
			wantStatus: 400,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "Accept: application/xml",
			method:     "GET",
			url:        "/api/v1/users",
			headers:    map[string]string{"Accept": "application/xml", "Authorization": "Bearer " + adminToken},
			wantStatus: 406,
			wantCode:   "NOT_ACCEPTABLE",
		},
		{
			name:       "Body gagal, aturan tag dilanggar",
			method:     "POST",
			url:        "/api/v1/auth/register",
			body:       `{"username":"a","email":"bad","password":"123"}`,
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: 422,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.url, bodyReader)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			resp, body, err := executeRequest(app, req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", resp.StatusCode, tc.wantStatus, body)
			}

			var errResp model.ErrorResponse
			if err := json.Unmarshal([]byte(body), &errResp); err != nil {
				t.Fatalf("JSON decode error: %v. Body: %s", err, body)
			}

			if errResp.Code != tc.wantCode {
				t.Errorf("Code = %s, want %s", errResp.Code, tc.wantCode)
			}
			if errResp.RequestID == "" {
				t.Errorf("Expected RequestID to be populated, got empty")
			}
			if errResp.Success != false {
				t.Errorf("Expected Success = false, got true")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test D.2, D.3, D.4: Students Entity Advanced Features
// -----------------------------------------------------------------------------
func TestStudentsAdvancedFeatures(t *testing.T) {
	app := setupTestApp(t)

	// D.2: Validasi Deklaratif Students
	// 1. NIM tidak sesuai format (bukan 11-12 digit)
	badNIM := `{"nim":"123","name":"Mahasiswa Uji","grade":85}`
	reqBadNIM := httptest.NewRequest("POST", "/api/v1/students", strings.NewReader(badNIM))
	reqBadNIM.Header.Set("Content-Type", "application/json")
	reqBadNIM.Header.Set("Authorization", "Bearer "+adminToken)

	respBadNIM, bodyBadNIM, _ := executeRequest(app, reqBadNIM)
	if respBadNIM.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422 for invalid NIM, got %d. Body: %s", respBadNIM.StatusCode, bodyBadNIM)
	}
	var errBadNIM model.ErrorResponse
	_ = json.Unmarshal([]byte(bodyBadNIM), &errBadNIM)
	if errBadNIM.Fields["nim"] != "format NIM tidak valid, harus 11-12 digit angka" {
		t.Errorf("Expected NIM format error, got: %v", errBadNIM.Fields["nim"])
	}

	// 2. PATCH student dengan name kosong: {"name":""} DITOLAK (422)
	patchEmptyName := `{"name":""}`
	reqPatchEmpty := httptest.NewRequest("PATCH", "/api/v1/students/6", strings.NewReader(patchEmptyName))
	reqPatchEmpty.Header.Set("Content-Type", "application/json")
	reqPatchEmpty.Header.Set("Authorization", "Bearer "+budiToken)

	respPatchEmpty, bodyPatchEmpty, _ := executeRequest(app, reqPatchEmpty)
	if respPatchEmpty.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422 for empty name patch on student, got %d. Body: %s", respPatchEmpty.StatusCode, bodyPatchEmpty)
	}

	// 3. PATCH student tanpa name (misal hanya ubah grade): {"grade":94} DITERIMA (200 OK)
	patchGradeOnly := `{"grade":94}`
	reqPatchGrade := httptest.NewRequest("PATCH", "/api/v1/students/6", strings.NewReader(patchGradeOnly))
	reqPatchGrade.Header.Set("Content-Type", "application/json")
	reqPatchGrade.Header.Set("Authorization", "Bearer "+budiToken)

	respPatchGrade, bodyPatchGrade, _ := executeRequest(app, reqPatchGrade)
	if respPatchGrade.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for valid patch omitting name, got %d. Body: %s", respPatchGrade.StatusCode, bodyPatchGrade)
	}

	// D.3: Keyset Cursor Pagination pada Students
	reqStudentsPage1 := httptest.NewRequest("GET", "/api/v1/students?limit=2", nil)
	reqStudentsPage1.Header.Set("Authorization", "Bearer "+adminToken)

	respStudentsP1, bodyStudentsP1, _ := executeRequest(app, reqStudentsPage1)
	if respStudentsP1.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for students list, got %d. Body: %s", respStudentsP1.StatusCode, bodyStudentsP1)
	}

	var jsonStudentsP1 model.WebCursorResponse
	_ = json.Unmarshal([]byte(bodyStudentsP1), &jsonStudentsP1)
	if jsonStudentsP1.Meta == nil || jsonStudentsP1.Meta.NextCursor == "" {
		t.Fatalf("Expected NextCursor for students list, got nil or empty")
	}

	// D.4: Content Negotiation pada Students (JSON vs CSV)
	reqStudentsCSV := httptest.NewRequest("GET", "/api/v1/students?limit=3", nil)
	reqStudentsCSV.Header.Set("Authorization", "Bearer "+adminToken)
	reqStudentsCSV.Header.Set("Accept", "text/csv")

	respStudentsCSV, bodyStudentsCSV, _ := executeRequest(app, reqStudentsCSV)
	if respStudentsCSV.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for students CSV, got %d. Body: %s", respStudentsCSV.StatusCode, bodyStudentsCSV)
	}
	if !strings.Contains(bodyStudentsCSV, "id,nim,name,grade,is_active,owner_id,created_at") {
		t.Fatalf("Students CSV header missing! Body: %q", bodyStudentsCSV)
	}
	t.Logf("Students CSV output:\n%s", bodyStudentsCSV)
}
