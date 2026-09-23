package service

import (
	"api-students/app/model"
	"api-students/helper"
)

// CanAccessStudent memutuskan apakah seseorang boleh mengakses atau memodifikasi data student tertentu.
//
// Fungsi ini adalah fungsi murni (pure function) dan sengaja tidak mengimpor Fiber maupun Repository
// agar mudah diuji secara unit testing tanpa ketergantungan framework.
//
// Dua jalur akses yang diizinkan:
// 1. Kepemilikan (ownership): ownerID sama dengan current.UserID (pembuat data berhak atas datanya).
// 2. Permission: Role pemanggil memiliki permission spesifik (:any) untuk mengakses data siapa pun.
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// Jalur 1: Kepemilikan (ownership)
	if ownerID > 0 && current.UserID == ownerID {
		return true
	}

	// Jalur 2: Permission berbasis role
	return perms.Can(current.Role, anyPermission)
}
