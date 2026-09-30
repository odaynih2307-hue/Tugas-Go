package model

import "time"

// Student merepresentasikan data mahasiswa di database.
type Student struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	NIM         string     `json:"nim"`
	Nama        string     `json:"nama"`
	Prodi       string     `json:"prodi"`
	Angkatan    int        `json:"angkatan"`
	IPKTerakhir *float64   `json:"ipk_terakhir"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// StudentListItem adalah format tampilan ringkas per mahasiswa pada endpoint GET /api/v1/students.
type StudentListItem struct {
	ID          int      `json:"id"`
	NIM         string   `json:"nim"`
	Nama        string   `json:"nama"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

// CreateStudentRequest merepresentasikan payload pendaftaran mahasiswa baru oleh admin.
type CreateStudentRequest struct {
	NIM         string   `json:"nim"`
	Nama        string   `json:"nama"`
	Email       string   `json:"email"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

// UpdateStudentRequest merepresentasikan payload perubahan data mahasiswa (tanpa NIM).
type UpdateStudentRequest struct {
	Nama        string   `json:"nama"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

// StudentDetailResponse adalah format respons untuk detail mahasiswa beserta total SKS dan daftar KRS.
type StudentDetailResponse struct {
	ID          int                   `json:"id"`
	NIM         string                `json:"nim"`
	Nama        string                `json:"nama"`
	Prodi       string                `json:"prodi"`
	Angkatan    int                   `json:"angkatan"`
	IPKTerakhir *float64              `json:"ipk_terakhir"`
	TotalSKS    int                   `json:"total_sks"`
	BatasSKS    int                   `json:"batas_sks"`
	MataKuliah  []EnrolledCourseItem `json:"mata_kuliah"`
}

// StudentQuery mewakili filter dan query parameter pada GET /api/v1/students.
type StudentQuery struct {
	Page     int
	PerPage  int
	Prodi    string
	Angkatan int
	Search   string
	Sort     string
}

func (q StudentQuery) Offset() int {
	if q.Page <= 1 {
		return 0
	}
	return (q.Page - 1) * q.PerPage
}
