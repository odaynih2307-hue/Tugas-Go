package model

import "time"

// Enrollment merepresentasikan pengambilan mata kuliah oleh mahasiswa (KRS).
type Enrollment struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateEnrollmentRequest merepresentasikan payload saat mahasiswa mengambil mata kuliah.
type CreateEnrollmentRequest struct {
	CourseID      int    `json:"course_id"`
	TahunAkademik string `json:"tahun_akademik"`
}

// EnrolledCourseItem menyajikan detail mata kuliah yang diambil mahasiswa.
type EnrolledCourseItem struct {
	EnrollmentID  int       `json:"enrollment_id"`
	CourseID      int       `json:"course_id"`
	KodeMK        string    `json:"kode_mk"`
	NamaMK        string    `json:"nama_mk"`
	SKS           int       `json:"sks"`
	Semester      int       `json:"semester"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
}
