package service

import (
	"strings"

	"api-students/app/model"
)

// File ini berisi business rules murni.
// Tidak menggunakan Fiber, tidak mengakses database,
// dan tidak mengetahui HTTP.

// ValidateCreate memeriksa data untuk pembuatan student.
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM wajib diisi"
	}

	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "nama wajib diisi"
	}

	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	return errs
}

// ValidateReplace memeriksa data untuk penggantian student melalui PUT.
// Seluruh field utama harus valid karena PUT mengganti data secara keseluruhan.
func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM wajib diisi"
	}

	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "nama wajib diisi"
	}

	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	return errs
}

// ApplyPatch menerapkan field yang dikirim ke data student yang sudah ada.
// Field yang nil tidak akan mengubah nilai sebelumnya.
func ApplyPatch(
	current model.Student,
	req model.PatchStudentRequest,
) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.NIM != nil {
		trimmed := strings.TrimSpace(*req.NIM)

		if trimmed == "" {
			errs["nim"] = "NIM tidak boleh kosong"
		} else {
			current.NIM = trimmed
		}
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)

		if trimmed == "" {
			errs["name"] = "nama tidak boleh kosong"
		} else {
			current.Name = trimmed
		}
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			errs["grade"] = "grade harus antara 0 dan 100"
		} else {
			current.Grade = *req.Grade
		}
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

// IsEmptyPatch memeriksa apakah PATCH tidak mengirimkan field apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil &&
		req.Name == nil &&
		req.Grade == nil &&
		req.IsActive == nil
}

// CountTotalPages menghitung jumlah halaman pagination.
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}

	return (total + limit - 1) / limit
}
