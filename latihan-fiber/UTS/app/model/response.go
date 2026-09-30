package model

// APIResponse merepresentasikan struktur respons seragam untuk respons sukses atau umum.
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

// ErrorResponse merepresentasikan respons error dengan validasi per kolom.
type ErrorResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

// SimpleErrorResponse merepresentasikan respons error umum tanpa per-field errors.
type SimpleErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Errors  any    `json:"errors"`
}

// Meta menyimpan informasi pagination sesuai spesifikasi PDF.
type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}
