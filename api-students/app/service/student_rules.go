package service

import (
	"strings"

	"api-students/app/model"
)

// ApplyPatchStudent menerapkan field PATCH ke entitas Student yang ada.
// Pemeriksaan format data sudah selesai ditangani secara deklaratif oleh tag
// validator sebelum fungsi ini dipanggil.
func ApplyPatchStudent(
	current model.Student,
	req model.PatchStudentRequest,
) model.Student {
	if req.NIM != nil {
		current.NIM = strings.TrimSpace(*req.NIM)
	}

	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}

	if req.Grade != nil {
		current.Grade = *req.Grade
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current
}

// IsEmptyPatchStudent memeriksa apakah request body PATCH kosong sama sekali.
func IsEmptyPatchStudent(req model.PatchStudentRequest) bool {
	return req.NIM == nil &&
		req.Name == nil &&
		req.Grade == nil &&
		req.IsActive == nil
}

// CountTotalPages menghitung jumlah halaman pagination offset (legacy).
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}

	return (total + limit - 1) / limit
}
