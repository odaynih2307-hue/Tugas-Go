package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

var (
	modul10Pool        *pgxpool.Pool
	modul10MongoClient *mongo.Client
	modul10MongoDB     *mongo.Database
	modul10JWT         *helper.JWTManager
)

func initModul10Env(t *testing.T) {
	_ = godotenv.Load("../.env")
	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	modul10JWT = helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		15*time.Minute,
	)

	var err error
	modul10Pool, err = database.NewPool(context.Background())
	if err != nil {
		t.Fatalf("Gagal terhubung database PG: %v", err)
	}

	modul10MongoClient, err = database.NewMongoClient(context.Background())
	if err != nil {
		t.Fatalf("Gagal terhubung MongoDB: %v", err)
	}
	modul10MongoDB = modul10MongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))
}

func getM10Token(t *testing.T, username string) string {
	var u model.User
	err := modul10Pool.QueryRow(context.Background(),
		"SELECT id, username, email, role, is_active FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		t.Fatalf("User %s tidak ditemukan: %v", username, err)
	}

	token, _, err := modul10JWT.GenerateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		t.Fatalf("Gagal generate token: %v", err)
	}
	return token
}

func doM10HTTP(t *testing.T, method, path, token string, body any) (int, map[string]any, http.Header) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBytes, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body error: %v", err)
			}
			bodyReader = bytes.NewBuffer(jsonBytes)
		}
	}

	req, err := http.NewRequest(method, "http://localhost:3000/api/v1"+path, bodyReader)
	if err != nil {
		t.Fatalf("new request error: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http call error: %v", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var res map[string]any
	_ = json.Unmarshal(respBytes, &res)

	return resp.StatusCode, res, resp.Header
}

func TestModul10_AllEndpointsAndScenarios(t *testing.T) {
	initModul10Env(t)
	defer modul10Pool.Close()

	// Bersihkan data ulasan & portofolio uji di MongoDB
	_, _ = modul10MongoDB.Collection("course_reviews").DeleteMany(context.Background(), bson.M{})
	_, _ = modul10MongoDB.Collection("student_portfolios").DeleteMany(context.Background(), bson.M{})

	adminToken := getM10Token(t, "admin")
	mhs1Token := getM10Token(t, "mhs001")
	mhs2Token := getM10Token(t, "mhs002")
	mhs3Token := getM10Token(t, "mhs003")

	var if101ID int
	_ = modul10Pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF101'").Scan(&if101ID)

	// Pastikan mhs001 dan mhs002 terdaftar di IF101, sedangkan mhs003 TIDAK
	var u1ID, u2ID int
	_ = modul10Pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = 'mhs001'").Scan(&u1ID)
	_ = modul10Pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = 'mhs002'").Scan(&u2ID)

	_, _ = modul10Pool.Exec(context.Background(), "DELETE FROM enrollments WHERE course_id = $1", if101ID)
	_, _ = modul10Pool.Exec(context.Background(), "INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2), ($1, $3)", if101ID, u1ID, u2ID)
	_, _ = modul10Pool.Exec(context.Background(), "UPDATE courses SET enrolled_count = 2 WHERE id = $1", if101ID)

	var reviewID string

	// 1. mhs003 menulis ulasan IF101 -> 403 NOT_ENROLLED
	t.Run("1. mhs003 menulis ulasan IF101 -> 403 NOT_ENROLLED", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", fmt.Sprintf("/courses/%d/reviews", if101ID), mhs3Token, map[string]any{
			"rating":  5,
			"comment": "Materi sangat menarik dan aplikatif",
			"tags":    []string{"seru", "rekomendasi"},
		})
		if status != http.StatusForbidden || res["code"] != "NOT_ENROLLED" {
			t.Fatalf("expected 403 NOT_ENROLLED, got %d: %v", status, res)
		}
	})

	// 2. mhs001 menulis ulasan dengan tag ["Tugas-Banyak", " dosen-jelas ", "tugas-banyak"] -> 201 Created & normalized tags
	t.Run("2. mhs001 menulis ulasan -> 201, tag dinormalisasi", func(t *testing.T) {
		status, res, header := doM10HTTP(t, "POST", fmt.Sprintf("/courses/%d/reviews", if101ID), mhs1Token, map[string]any{
			"rating":  5,
			"comment": "Materi Go-nya jelas, tugasnya cukup banyak",
			"tags":    []string{"Tugas-Banyak", " dosen-jelas ", "tugas-banyak"},
		})
		if status != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %v", status, res)
		}
		loc := header.Get("Location")
		if loc == "" {
			t.Fatalf("expected Location header to be set, got empty")
		}
		data := res["data"].(map[string]any)
		reviewID = data["id"].(string)
		tags := data["tags"].([]any)
		if len(tags) != 2 || tags[0] != "tugas-banyak" || tags[1] != "dosen-jelas" {
			t.Fatalf("expected tags ['tugas-banyak', 'dosen-jelas'], got %v", tags)
		}
	})

	// 3. mhs001 menulis ulasan kedua di kelas yang sama -> 409 ALREADY_REVIEWED
	t.Run("3. mhs001 menulis ulasan kedua -> 409 ALREADY_REVIEWED", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", fmt.Sprintf("/courses/%d/reviews", if101ID), mhs1Token, map[string]any{
			"rating":  4,
			"comment": "Ulasan kedua saya coba kirim kembali",
			"tags":    []string{"seru"},
		})
		if status != http.StatusConflict || res["code"] != "ALREADY_REVIEWED" {
			t.Fatalf("expected 409 ALREADY_REVIEWED, got %d: %v", status, res)
		}
	})

	// 4. Rating 9, komentar "pendek", 6 tag -> 422 VALIDATION_ERROR
	t.Run("4. Input tidak valid -> 422 VALIDATION_ERROR", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", fmt.Sprintf("/courses/%d/reviews", if101ID), mhs2Token, map[string]any{
			"rating":  9,
			"comment": "pendek",
			"tags":    []string{"t1", "t2", "t3", "t4", "t5", "t6"},
		})
		if status != http.StatusUnprocessableEntity || res["code"] != "VALIDATION_ERROR" {
			t.Fatalf("expected 422 VALIDATION_ERROR, got %d: %v", status, res)
		}
	})

	// 5. Ulasan untuk kelas 99999 -> 404
	t.Run("5. Ulasan untuk kelas 99999 -> 404", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", "/courses/99999/reviews", mhs1Token, map[string]any{
			"rating":  4,
			"comment": "Ulasan untuk kelas yang tidak ada",
			"tags":    []string{"seru"},
		})
		if status != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %v", status, res)
		}
	})

	// 6. GET /reviews/bukan-objectid -> 400 "id ulasan tidak valid"
	t.Run("6. GET /reviews/bukan-objectid -> 400", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "GET", "/reviews/bukan-objectid", mhs1Token, nil)
		if status != http.StatusBadRequest || res["code"] != "BAD_REQUEST" {
			t.Fatalf("expected 400 BAD_REQUEST, got %d: %v", status, res)
		}
	})

	// 7. mhs002 mengubah ulasan mhs001 -> 403 FORBIDDEN
	t.Run("7. mhs002 mengubah ulasan mhs001 -> 403 FORBIDDEN", func(t *testing.T) {
		newRating := 3
		status, res, _ := doM10HTTP(t, "PATCH", "/reviews/"+reviewID, mhs2Token, map[string]any{
			"rating": &newRating,
		})
		if status != http.StatusForbidden {
			t.Fatalf("expected 403 FORBIDDEN, got %d: %v", status, res)
		}
	})

	// 8. mhs001 mengubah ulasannya sendiri -> 200, updated_at berubah
	t.Run("8. mhs001 mengubah ulasannya sendiri -> 200 OK", func(t *testing.T) {
		newComment := "Materi sangat jelas dan praktikum menantang"
		status, res, _ := doM10HTTP(t, "PATCH", "/reviews/"+reviewID, mhs1Token, map[string]any{
			"comment": &newComment,
		})
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %v", status, res)
		}
	})

	// 9. mhs001 menilai ulasannya sendiri "membantu" -> 409 OWN_REVIEW
	t.Run("9. mhs001 menilai ulasannya sendiri -> 409 OWN_REVIEW", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", "/reviews/"+reviewID+"/helpful", mhs1Token, nil)
		if status != http.StatusConflict || res["code"] != "OWN_REVIEW" {
			t.Fatalf("expected 409 OWN_REVIEW, got %d: %v", status, res)
		}
	})

	// 10. mhs002 menilai "membantu", lalu mengulanginya -> 200 lalu 409 ALREADY_VOTED
	t.Run("10. mhs002 menilai membantu -> 200 lalu 409 ALREADY_VOTED", func(t *testing.T) {
		status1, res1, _ := doM10HTTP(t, "POST", "/reviews/"+reviewID+"/helpful", mhs2Token, nil)
		if status1 != http.StatusOK {
			t.Fatalf("expected 200 OK first vote, got %d: %v", status1, res1)
		}

		status2, res2, _ := doM10HTTP(t, "POST", "/reviews/"+reviewID+"/helpful", mhs2Token, nil)
		if status2 != http.StatusConflict || res2["code"] != "ALREADY_VOTED" {
			t.Fatalf("expected 409 ALREADY_VOTED second vote, got %d: %v", status2, res2)
		}
	})

	// 11. Balasan ulasan (Reply): mhs002 membalas ulasan -> 201 Created
	t.Run("11. mhs002 membalas ulasan -> 201 Created", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", "/reviews/"+reviewID+"/replies", mhs2Token, map[string]any{
			"body": "Setuju sekali, tugasnya memang cukup banyak tapi seru",
		})
		if status != http.StatusCreated {
			t.Fatalf("expected 201 Created reply, got %d: %v", status, res)
		}
	})

	// 12. Ringkasan Aggregation: GET /courses/:id/reviews/summary -> 200 OK
	t.Run("12. Ringkasan ulasan (Aggregation) -> 200 OK", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "GET", fmt.Sprintf("/courses/%d/reviews/summary", if101ID), mhs1Token, nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK summary, got %d: %v", status, res)
		}
		data := res["data"].(map[string]any)
		if data["count"].(float64) != 1 {
			t.Fatalf("expected count 1, got %v", data["count"])
		}
	})

	// 13. Admin menghapus ulasan mhs001 (review:moderate) -> 204 No Content
	t.Run("13. Admin menghapus ulasan (review:moderate) -> 204 No Content", func(t *testing.T) {
		status, _, _ := doM10HTTP(t, "DELETE", "/reviews/"+reviewID, adminToken, nil)
		if status != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", status)
		}

		// Menghapus ulasan yang sudah terhapus -> 404
		delAgain, _, _ := doM10HTTP(t, "DELETE", "/reviews/"+reviewID, adminToken, nil)
		if delAgain != http.StatusNotFound {
			t.Fatalf("expected 404 on deleted review, got %d", delAgain)
		}
	})

	// 14. NoSQL injection ditolak
	t.Run("14. NoSQL injection ditolak di body -> 400 Bad Request", func(t *testing.T) {
		rawBody := `{"comment": {"$ne": ""}}`
		status, _, _ := doM10HTTP(t, "POST", fmt.Sprintf("/courses/%d/reviews", if101ID), mhs1Token, rawBody)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for NoSQL injection object in string field, got %d", status)
		}
	})

	// -------------------------------------------------------------
	// BAGIAN C: PENGUJIAN FITUR PORTOFOLIO MAHASISWA
	// -------------------------------------------------------------

	// Ambil ID mahasiswa nyata dari tabel students
	var validStudentID int
	_ = modul10Pool.QueryRow(context.Background(), "SELECT id FROM students ORDER BY id LIMIT 1").Scan(&validStudentID)

	// 15. Portofolio untuk mahasiswa tidak dikenal -> 404
	t.Run("15. Portofolio mahasiswa tidak dikenal -> 404", func(t *testing.T) {
		status, _, _ := doM10HTTP(t, "GET", "/students/99999/portfolio", adminToken, nil)
		if status != http.StatusNotFound {
			t.Fatalf("expected 404 student not found in PG, got %d", status)
		}
	})

	// 16. PUT /students/:id/portfolio -> 200 OK (buat atau ganti portofolio)
	t.Run("16. PUT /students/:id/portfolio -> 200 OK", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "PUT", fmt.Sprintf("/students/%d/portfolio", validStudentID), adminToken, map[string]any{
			"headline": "Fullstack Go & Cloud Developer",
			"bio":      "Mahasiswa D4 Teknik Informatika UNAIR dengan minat pada sistem terdistribusi",
		})
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %v", status, res)
		}
		data := res["data"].(map[string]any)
		if data["student_id"].(float64) != float64(validStudentID) {
			t.Fatalf("expected student_id %d, got %v", validStudentID, data["student_id"])
		}
	})

	// 17. GET /students/:id/portfolio -> 200 OK
	t.Run("17. GET /students/:id/portfolio -> 200 OK", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "GET", fmt.Sprintf("/students/%d/portfolio", validStudentID), adminToken, nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %v", status, res)
		}
	})

	var projectID string
	// 18. POST /students/:id/portfolio/projects -> 201 Created
	t.Run("18. POST /students/:id/portfolio/projects -> 201 Created", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "POST", fmt.Sprintf("/students/%d/portfolio/projects", validStudentID), adminToken, map[string]any{
			"title":        "Sistem SIAKAD Terdistribusi",
			"technologies": []string{"Golang", "PostgreSQL", "Docker", "golang"},
			"link":         "https://github.com/odaynih2307-hue/siakad-dist",
			"year":         2026,
		})
		if status != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %v", status, res)
		}
		data := res["data"].(map[string]any)
		projects := data["projects"].([]any)
		if len(projects) == 0 {
			t.Fatalf("expected at least 1 project in portfolio")
		}
		p0 := projects[0].(map[string]any)
		projectID = p0["id"].(string)
		techs := p0["technologies"].([]any)
		// Harus ternormalisasi: "golang", "postgresql", "docker" (tanpa duplikat)
		if len(techs) != 3 {
			t.Fatalf("expected 3 normalized technologies, got %v", techs)
		}
	})

	// 19. GET /portfolios/stats/technologies -> 200 OK (Aggregation dengan 2x $unwind)
	t.Run("19. GET /portfolios/stats/technologies -> 200 OK", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "GET", "/portfolios/stats/technologies", adminToken, nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK stats, got %d: %v", status, res)
		}
		data := res["data"].([]any)
		if len(data) == 0 {
			t.Fatalf("expected stats array, got empty")
		}
	})

	// 20. DELETE /students/:id/portfolio/projects/:projectId -> 200 OK ($pull)
	t.Run("20. DELETE /students/:id/portfolio/projects/:projectId -> 200 OK", func(t *testing.T) {
		status, res, _ := doM10HTTP(t, "DELETE", fmt.Sprintf("/students/%d/portfolio/projects/%s", validStudentID, projectID), adminToken, nil)
		if status != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %v", status, res)
		}
		data := res["data"].(map[string]any)
		projects := data["projects"].([]any)
		if len(projects) != 0 {
			t.Fatalf("expected 0 projects after deletion, got %d", len(projects))
		}
	})

	// 21. Uji Konkurensi Batas 20 Proyek (C.3): Isi 18 proyek, kirim 10 serentak -> tepat 2 sukses, 8 ditolak 409
	t.Run("21. Uji Beban Batas 20 Proyek Serentak (18 + 10 -> tepat 2 sukses)", func(t *testing.T) {
		// Isi portofolio hingga tepat 18 proyek
		for i := 1; i <= 18; i++ {
			st, _, _ := doM10HTTP(t, "POST", fmt.Sprintf("/students/%d/portfolio/projects", validStudentID), adminToken, map[string]any{
				"title":        fmt.Sprintf("Proyek Batch A #%02d", i),
				"technologies": []string{"golang", "mongodb"},
				"link":         fmt.Sprintf("https://github.com/odaynih2307-hue/proj-%d", i),
				"year":         2026,
			})
			if st != http.StatusCreated {
				t.Fatalf("gagal mengisi proyek ke-%d: status %d", i, st)
			}
		}

		// Kirim 10 request serentak
		var wg sync.WaitGroup
		results := make(chan int, 10)
		startGate := make(chan struct{})

		for i := 1; i <= 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-startGate
				st, _, _ := doM10HTTP(t, "POST", fmt.Sprintf("/students/%d/portfolio/projects", validStudentID), adminToken, map[string]any{
					"title":        fmt.Sprintf("Proyek Serentak #%02d", idx),
					"technologies": []string{"golang", "react"},
					"link":         fmt.Sprintf("https://github.com/odaynih2307-hue/proj-concurrent-%d", idx),
					"year":         2026,
				})
				results <- st
			}(i)
		}

		close(startGate)
		wg.Wait()
		close(results)

		successCount := 0
		rejectedCount := 0
		for st := range results {
			if st == http.StatusCreated {
				successCount++
			} else if st == http.StatusConflict {
				rejectedCount++
			}
		}

		if successCount != 2 || rejectedCount != 8 {
			t.Fatalf("expected tepat 2 sukses dan 8 ditolak, got %d sukses dan %d ditolak", successCount, rejectedCount)
		}

		// Verifikasi total proyek di portofolio tepat 20
		_, getRes, _ := doM10HTTP(t, "GET", fmt.Sprintf("/students/%d/portfolio", validStudentID), adminToken, nil)
		doc := getRes["data"].(map[string]any)
		totalProjs := len(doc["projects"].([]any))
		if totalProjs != 20 {
			t.Fatalf("expected total 20 projects in document, got %d", totalProjs)
		}
	})
}
