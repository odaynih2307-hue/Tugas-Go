//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"api-students/app/model"
)

// swap mengirim POST /enrollments/swap atas nama student dan
// mengembalikan status code beserta field "code" pada response.
func (a api) swap(t *testing.T, student model.User, fromCourseID, toCourseID int) (int, string) {
	t.Helper()

	token, _, err := a.jwt.GenerateAccessToken(student.ID, student.Username, "user")
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}

	payload, _ := json.Marshal(model.SwapRequest{
		FromCourseID: fromCourseID,
		ToCourseID:   toCourseID,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request swap: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

// drop mengirim DELETE /courses/:id/enroll atas nama student dan
// mengembalikan status code.
func (a api) drop(t *testing.T, student model.User, courseID int) int {
	t.Helper()

	token, _, err := a.jwt.GenerateAccessToken(student.ID, student.Username, "user")
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request drop: %v", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

// TestSwapEndpoint_TargetCourseFull menguji penolakan perpindahan kelas (swap)
// ketika kelas tujuan sudah penuh (Tugas Mandiri D.2 butir 1).
// Memastikan:
// 1. Response HTTP 409 dengan code COURSE_FULL
// 2. Mahasiswa tetap terdaftar di kelas asal
// 3. enrolled_count pada kedua kelas tidak berubah
// 4. Tidak ada event enrollment.swapped di tabel outbox
func TestSwapEndpoint_TargetCourseFull(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	otherStudent := seedStudent(t)

	// Kelas asal: kuota 40, mahasiswa terdaftar
	courseOrigin := seedCourse(t, 3, 40)
	if status, _ := a.enroll(t, student, courseOrigin.ID); status != http.StatusCreated {
		t.Fatalf("enroll kelas asal: status = %d, seharusnya 201", status)
	}

	// Kelas tujuan: kuota 1, dipenuhi oleh otherStudent
	courseTarget := seedCourse(t, 3, 1)
	if status, _ := a.enroll(t, otherStudent, courseTarget.ID); status != http.StatusCreated {
		t.Fatalf("enroll kelas tujuan oleh other: status = %d, seharusnya 201", status)
	}

	// Simpan outbox count sebelum swap
	outboxBefore := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1`,
		model.EventEnrollmentSwapped)

	// Lakukan aksi swap dari courseOrigin ke courseTarget yang sudah penuh
	status, code := a.swap(t, student, courseOrigin.ID, courseTarget.ID)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Errorf("status = %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	// 1. Mahasiswa tetap terdaftar di kelas asal
	enrolledOrigin := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		courseOrigin.ID, student.ID)
	if enrolledOrigin != 1 {
		t.Errorf("mahasiswa harus tetap terdaftar di kelas asal (got %d, want 1)", enrolledOrigin)
	}

	// 2. Mahasiswa TIDAK terdaftar di kelas tujuan
	enrolledTarget := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		courseTarget.ID, student.ID)
	if enrolledTarget != 0 {
		t.Errorf("mahasiswa tidak boleh terdaftar di kelas tujuan (got %d, want 0)", enrolledTarget)
	}

	// 3. enrolled_count kedua kelas tidak berubah
	originCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, courseOrigin.ID)
	if originCount != 1 {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap 1", originCount)
	}
	targetCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, courseTarget.ID)
	if targetCount != 1 {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap 1", targetCount)
	}

	// 4. Tidak ada event enrollment.swapped di outbox
	outboxAfter := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1`,
		model.EventEnrollmentSwapped)
	if outboxAfter != outboxBefore {
		t.Errorf("tidak boleh ada event enrollment.swapped di outbox saat swap gagal (before: %d, after: %d)",
			outboxBefore, outboxAfter)
	}
}

// TestDropEndpoint_SuccessAndOutboxEvent menguji pembatalan pendaftaran kelas (drop)
// (Tugas Mandiri D.2 butir 2).
// Memastikan:
// 1. Response HTTP 204 No Content
// 2. Baris pendaftaran di tabel enrollments terhapus
// 3. enrolled_count berkurang satu
// 4. Outbox mencatat tepat satu event enrollment.dropped
func TestDropEndpoint_SuccessAndOutboxEvent(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	course := seedCourse(t, 3, 40)

	// Enroll terlebih dahulu
	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("enroll kelas: status = %d, seharusnya 201", status)
	}

	// Verifikasi pendaftaran aktif
	if countRows(t, `SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`, course.ID, student.ID) != 1 {
		t.Fatalf("mahasiswa belum terdaftar")
	}

	// Lakukan DELETE /courses/:id/enroll
	status := a.drop(t, student, course.ID)
	if status != http.StatusNoContent {
		t.Errorf("status drop = %d, seharusnya 204", status)
	}

	// 1. Baris enrollment hilang
	if n := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`, course.ID, student.ID); n != 0 {
		t.Errorf("baris enrollment = %d, seharusnya 0 setelah drop", n)
	}

	// 2. enrolled_count berkurang menjadi 0
	if c := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID); c != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0 setelah drop", c)
	}

	// 3. Outbox mencatat event enrollment.dropped
	dropEvents := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped)
	if dropEvents != 1 {
		t.Errorf("event enrollment.dropped di outbox = %d, seharusnya tepat 1", dropEvents)
	}
}
