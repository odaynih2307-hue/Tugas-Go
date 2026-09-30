package model

import "time"

// Course merepresentasikan model mata kuliah di database.
type Course struct {
	ID        int       `json:"id"`
	KodeMK    string    `json:"kode_mk"`
	NamaMK    string    `json:"nama_mk"`
	SKS       int       `json:"sks"`
	Semester  int       `json:"semester"`
	Kuota     int       `json:"kuota"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseResponse adalah format respons daftar mata kuliah beserta terisi dan sisa kuota.
type CourseResponse struct {
	ID        int    `json:"id"`
	KodeMK    string `json:"kode_mk"`
	NamaMK    string `json:"nama_mk"`
	SKS       int    `json:"sks"`
	Semester  int    `json:"semester"`
	Kuota     int    `json:"kuota"`
	Terisi    int    `json:"terisi"`
	SisaKuota int    `json:"sisa_kuota"`
}

// CourseQuery menyimpan parameter filter untuk GET /api/v1/courses.
type CourseQuery struct {
	Semester  int
	Search    string
	Available bool
}
