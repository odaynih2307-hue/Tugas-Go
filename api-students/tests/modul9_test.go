package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

var (
	testBaseURL = "http://localhost:3000/api/v1"
	modul9Pool  *pgxpool.Pool
	modul9JWT   *helper.JWTManager
)

func initTestEnv(t *testing.T) {
	_ = godotenv.Load("../.env")
	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	modul9JWT = helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		15*time.Minute,
	)

	var err error
	modul9Pool, err = database.NewPool(context.Background())
	if err != nil {
		t.Fatalf("Gagal terhubung database: %v", err)
	}
}

func getTestToken(t *testing.T, username string) string {
	var u model.User
	err := modul9Pool.QueryRow(context.Background(),
		"SELECT id, username, email, role, is_active FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		t.Fatalf("User %s tidak ditemukan: %v", username, err)
	}

	token, _, err := modul9JWT.GenerateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		t.Fatalf("Gagal generate token: %v", err)
	}
	return token
}

func doHTTP(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body error: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, testBaseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request error: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http call error: %v", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var res map[string]any
	_ = json.Unmarshal(respBytes, &res)

	return resp.StatusCode, res
}

func resetDB(t *testing.T) {
	_, _ = modul9Pool.Exec(context.Background(),
		"DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'mhs%')")
	_, _ = modul9Pool.Exec(context.Background(),
		"UPDATE courses c SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)")
	_, _ = modul9Pool.Exec(context.Background(), "DELETE FROM supervisions")
	_, _ = modul9Pool.Exec(context.Background(),
		"UPDATE supervisors s SET supervised_count = (SELECT COUNT(*) FROM supervisions sp WHERE sp.supervisor_id = s.id)")
}

func TestModul9_FunctionalScenarios(t *testing.T) {
	initTestEnv(t)
	defer modul9Pool.Close()

	resetDB(t)

	adminToken := getTestToken(t, "admin")
	mhs1Token := getTestToken(t, "mhs001")
	mhs2Token := getTestToken(t, "mhs002")

	var if101ID, if102ID, if399ID int
	_ = modul9Pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF101'").Scan(&if101ID)
	_ = modul9Pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF102'").Scan(&if102ID)
	_ = modul9Pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF399'").Scan(&if399ID)

	// 1. Skenario: Daftar ke kelas yang masih kosong -> 201
	t.Run("1. Daftar ke kelas yang masih kosong -> 201", func(t *testing.T) {
		status, res := doHTTP(t, "POST", fmt.Sprintf("/courses/%d/enroll", if101ID), mhs1Token, nil)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %v", status, res)
		}
	})

	// 2. Skenario: Daftar ke kelas yang sama sekali lagi -> 409 ALREADY_ENROLLED
	t.Run("2. Daftar ke kelas yang sama sekali lagi -> 409 ALREADY_ENROLLED", func(t *testing.T) {
		status, res := doHTTP(t, "POST", fmt.Sprintf("/courses/%d/enroll", if101ID), mhs1Token, nil)
		if status != http.StatusConflict || res["code"] != "ALREADY_ENROLLED" {
			t.Fatalf("expected 409 ALREADY_ENROLLED, got %d: %v", status, res)
		}
	})

	// 3. Skenario: Mahasiswa ke-6 mendaftar ke IF399 (kuota 5) -> 409 COURSE_FULL (bukan 500)
	t.Run("3. Mahasiswa ke-6 mendaftar ke IF399 (kuota 5) -> 409 COURSE_FULL", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			tok := getTestToken(t, fmt.Sprintf("mhs%03d", i))
			st, r := doHTTP(t, "POST", fmt.Sprintf("/courses/%d/enroll", if399ID), tok, nil)
			if st != http.StatusCreated {
				t.Fatalf("mhs%03d enroll expected 201, got %d: %v", i, st, r)
			}
		}
		tok6 := getTestToken(t, "mhs006")
		status, res := doHTTP(t, "POST", fmt.Sprintf("/courses/%d/enroll", if399ID), tok6, nil)
		if status != http.StatusConflict || res["code"] != "COURSE_FULL" {
			t.Fatalf("expected 409 COURSE_FULL, got %d: %v", status, res)
		}
	})

	// 4. Skenario: Batalkan kelas yang tidak pernah diambil -> 409 NOT_ENROLLED
	t.Run("4. Batalkan kelas yang tidak pernah diambil -> 409 NOT_ENROLLED", func(t *testing.T) {
		status, res := doHTTP(t, "DELETE", fmt.Sprintf("/courses/%d/enroll", if102ID), mhs1Token, nil)
		if status != http.StatusConflict || res["code"] != "NOT_ENROLLED" {
			t.Fatalf("expected 409 NOT_ENROLLED, got %d: %v", status, res)
		}
	})

	// 5. Skenario: Pindah ke kelas yang penuh -> 409 COURSE_FULL, tetap di kelas asal
	t.Run("5. Pindah ke kelas yang penuh -> 409 COURSE_FULL", func(t *testing.T) {
		status, res := doHTTP(t, "POST", "/enrollments/swap", mhs1Token, map[string]int{
			"from_course_id": if101ID,
			"to_course_id":   if399ID,
		})
		if status != http.StatusConflict || res["code"] != "COURSE_FULL" {
			t.Fatalf("expected 409 COURSE_FULL, got %d: %v", status, res)
		}
	})

	// 6. Skenario: Pindah ke kelas yang masih kosong -> 200 OK
	t.Run("6. Pindah ke kelas yang masih kosong -> 200 OK", func(t *testing.T) {
		status, res := doHTTP(t, "POST", "/enrollments/swap", mhs1Token, map[string]int{
			"from_course_id": if101ID,
			"to_course_id":   if102ID,
		})
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %v", status, res)
		}
	})

	// 7. Skenario: Pindah dengan from_course_id == to_course_id -> 422 VALIDATION_ERROR
	t.Run("7. Pindah dengan from_course_id == to_course_id -> 422 VALIDATION_ERROR", func(t *testing.T) {
		status, res := doHTTP(t, "POST", "/enrollments/swap", mhs1Token, map[string]int{
			"from_course_id": if102ID,
			"to_course_id":   if102ID,
		})
		if status != http.StatusUnprocessableEntity || res["code"] != "VALIDATION_ERROR" {
			t.Fatalf("expected 422 VALIDATION_ERROR, got %d: %v", status, res)
		}
	})

	// 8. Skenario: Admin ubah kuota dengan version terbaru -> 200; version naik 1
	var latestVersion int
	t.Run("8. Admin ubah kuota dengan version terbaru -> 200", func(t *testing.T) {
		var curVer int
		_ = modul9Pool.QueryRow(context.Background(), "SELECT version FROM courses WHERE id = $1", if101ID).Scan(&curVer)
		status, res := doHTTP(t, "PATCH", fmt.Sprintf("/courses/%d", if101ID), adminToken, map[string]int{
			"quota":   45,
			"version": curVer,
		})
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %v", status, res)
		}
		data := res["data"].(map[string]any)
		latestVersion = int(data["version"].(float64))
		if latestVersion != curVer+1 {
			t.Fatalf("expected version %d, got %d", curVer+1, latestVersion)
		}
	})

	// 9. Skenario: Admin ubah kuota dengan version lama -> 409 VERSION_CONFLICT
	t.Run("9. Admin ubah kuota dengan version lama -> 409 VERSION_CONFLICT", func(t *testing.T) {
		status, res := doHTTP(t, "PATCH", fmt.Sprintf("/courses/%d", if101ID), adminToken, map[string]int{
			"quota":   50,
			"version": latestVersion - 1,
		})
		if status != http.StatusConflict || res["code"] != "VERSION_CONFLICT" {
			t.Fatalf("expected 409 VERSION_CONFLICT, got %d: %v", status, res)
		}
	})

	// 10. Skenario: Admin menurunkan kuota di bawah jumlah peserta -> 409 QUOTA_BELOW_ENROLLED
	t.Run("10. Admin menurunkan kuota di bawah jumlah peserta -> 409 QUOTA_BELOW_ENROLLED", func(t *testing.T) {
		var if399Ver int
		_ = modul9Pool.QueryRow(context.Background(), "SELECT version FROM courses WHERE id = $1", if399ID).Scan(&if399Ver)
		status, res := doHTTP(t, "PATCH", fmt.Sprintf("/courses/%d", if399ID), adminToken, map[string]int{
			"quota":   4, // smaller than enrolled_count 5
			"version": if399Ver,
		})
		if status != http.StatusConflict || res["code"] != "QUOTA_BELOW_ENROLLED" {
			t.Fatalf("expected 409 QUOTA_BELOW_ENROLLED, got %d: %v", status, res)
		}
	})

	// 11. Skenario: Mahasiswa mencoba mengubah kuota -> 403 FORBIDDEN
	t.Run("11. Mahasiswa mencoba mengubah kuota -> 403 FORBIDDEN", func(t *testing.T) {
		status, _ := doHTTP(t, "PATCH", fmt.Sprintf("/courses/%d", if101ID), mhs2Token, map[string]int{
			"quota":   40,
			"version": 1,
		})
		if status != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", status)
		}
	})

	// 12. Skenario: D.2 Supervisors Assign, Switch, Update Quota
	t.Run("12. Supervisors D.2 Assign, Duplicate, Switch, Quota", func(t *testing.T) {
		var validStudentID int
		err := modul9Pool.QueryRow(context.Background(), "SELECT id FROM students ORDER BY id LIMIT 1").Scan(&validStudentID)
		if err != nil {
			t.Fatalf("Gagal mengambil student id: %v", err)
		}

		// Assign student to supervisor 1
		st1, res1 := doHTTP(t, "POST", "/supervisors/1/assign", adminToken, map[string]int{"student_id": validStudentID})
		if st1 != http.StatusCreated {
			t.Fatalf("expected 201 assign supervisor, got %d: %v", st1, res1)
		}

		// Re-assign same student -> 409 ALREADY_SUPERVISED
		stDup, resDup := doHTTP(t, "POST", "/supervisors/2/assign", adminToken, map[string]int{"student_id": validStudentID})
		if stDup != http.StatusConflict || resDup["code"] != "ALREADY_SUPERVISED" {
			t.Fatalf("expected 409 ALREADY_SUPERVISED, got %d: %v", stDup, resDup)
		}

		// Switch student from supervisor 1 to supervisor 2 -> 200 OK
		stSwitch, resSwitch := doHTTP(t, "POST", "/supervisions/switch", adminToken, map[string]int{
			"student_id":       validStudentID,
			"to_supervisor_id": 2,
		})
		if stSwitch != http.StatusOK {
			t.Fatalf("expected 200 switch supervisor, got %d: %v", stSwitch, resSwitch)
		}

		// Optimistic locking on supervisor quota
		var supVer int
		_ = modul9Pool.QueryRow(context.Background(), "SELECT version FROM supervisors WHERE id = 1").Scan(&supVer)
		stPatch, resPatch := doHTTP(t, "PATCH", "/supervisors/1", adminToken, map[string]int{
			"quota":   6,
			"version": supVer,
		})
		if stPatch != http.StatusOK {
			t.Fatalf("expected 200 patch supervisor quota, got %d: %v", stPatch, resPatch)
		}
		data := resPatch["data"].(map[string]any)
		if int(data["version"].(float64)) != supVer+1 {
			t.Fatalf("expected supervisor version %d, got %v", supVer+1, data["version"])
		}
	})
}
