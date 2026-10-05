package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateStudentRequest memuat aturan validasi deklaratif (Tugas D.2).
type CreateStudentRequest struct {
	NIM      string  `json:"nim" validate:"required,validnim"`
	Name     string  `json:"name" validate:"required,min=3,max=100"`
	Grade    float64 `json:"grade" validate:"min=0,max=100"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// ReplaceStudentRequest memuat aturan validasi deklaratif untuk PUT.
type ReplaceStudentRequest struct {
	NIM      string  `json:"nim" validate:"required,validnim"`
	Name     string  `json:"name" validate:"required,min=3,max=100"`
	Grade    float64 `json:"grade" validate:"min=0,max=100"`
	IsActive bool    `json:"is_active"`
}

// PatchStudentRequest menggunakan pointer dan tag omitnil (Tugas D.2 butir 3).
// omitnil menjamin jika field tidak dikirim (nil) maka dilewati, tetapi jika
// dikirim bernilai kosong seperti {"name":""} maka TETAP diperiksa dan ditolak (422).
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty" validate:"omitnil,validnim"`
	Name     *string  `json:"name,omitempty" validate:"omitnil,min=3,max=100"`
	Grade    *float64 `json:"grade,omitempty" validate:"omitnil,min=0,max=100"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type WebCursorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    any         `json:"data,omitempty"`
	Meta    *CursorMeta `json:"meta,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
}

// Offset menghitung berapa baris yang dilewati untuk halaman ini.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}
