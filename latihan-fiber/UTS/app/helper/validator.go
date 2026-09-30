package helper

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"uts-siakad/app/model"
)

var (
	nimRegex           = regexp.MustCompile(`^\d{12}$`)
	tahunAkademikRegex = regexp.MustCompile(`^\d{4}/\d{4}-(Ganjil|Genap|Antara)$`)
)

// ValidateLoginRequest memvalidasi input email dan password pada login.
func ValidateLoginRequest(req model.LoginRequest) map[string][]string {
	errs := make(map[string][]string)

	if req.Email == "" {
		errs["email"] = append(errs["email"], "Email wajib diisi")
	} else if _, err := mail.ParseAddress(req.Email); err != nil {
		errs["email"] = append(errs["email"], "Format email tidak valid")
	}

	if req.Password == "" {
		errs["password"] = append(errs["password"], "Password wajib diisi")
	} else if len(req.Password) < 8 {
		errs["password"] = append(errs["password"], "Password minimal 8 karakter")
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateCreateStudentRequest memvalidasi input pendaftaran mahasiswa baru.
func ValidateCreateStudentRequest(req model.CreateStudentRequest) map[string][]string {
	errs := make(map[string][]string)
	currentYear := time.Now().Year()

	// NIM: wajib, 12 digit numerik
	if req.NIM == "" {
		errs["nim"] = append(errs["nim"], "NIM wajib diisi")
	} else if !nimRegex.MatchString(req.NIM) {
		errs["nim"] = append(errs["nim"], "NIM harus berupa 12 digit angka")
	}

	// Nama: wajib
	if req.Nama == "" {
		errs["nama"] = append(errs["nama"], "Nama wajib diisi")
	}

	// Email: wajib, format email
	if req.Email == "" {
		errs["email"] = append(errs["email"], "Email wajib diisi")
	} else if _, err := mail.ParseAddress(req.Email); err != nil {
		errs["email"] = append(errs["email"], "Format email tidak valid")
	}

	// Prodi: wajib
	if req.Prodi == "" {
		errs["prodi"] = append(errs["prodi"], "Program studi wajib diisi")
	}

	// Angkatan: wajib, 4 digit, <= tahun berjalan
	if req.Angkatan == 0 {
		errs["angkatan"] = append(errs["angkatan"], "Angkatan wajib diisi")
	} else if req.Angkatan < 1900 || req.Angkatan > currentYear {
		errs["angkatan"] = append(errs["angkatan"], fmt.Sprintf("Angkatan harus berupa 4 digit tahun (maksimal %d)", currentYear))
	}

	// IPK Terakhir: opsional, jika diisi rentang 0.00 s/d 4.00
	if req.IPKTerakhir != nil {
		if *req.IPKTerakhir < 0.00 || *req.IPKTerakhir > 4.00 {
			errs["ipk_terakhir"] = append(errs["ipk_terakhir"], "IPK terakhir harus berada dalam rentang 0.00 - 4.00")
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateUpdateStudentRequest memvalidasi input pembaruan mahasiswa (tanpa mengubah NIM).
func ValidateUpdateStudentRequest(req model.UpdateStudentRequest) map[string][]string {
	errs := make(map[string][]string)
	currentYear := time.Now().Year()

	if req.Nama == "" {
		errs["nama"] = append(errs["nama"], "Nama wajib diisi")
	}

	if req.Prodi == "" {
		errs["prodi"] = append(errs["prodi"], "Program studi wajib diisi")
	}

	if req.Angkatan == 0 {
		errs["angkatan"] = append(errs["angkatan"], "Angkatan wajib diisi")
	} else if req.Angkatan < 1900 || req.Angkatan > currentYear {
		errs["angkatan"] = append(errs["angkatan"], fmt.Sprintf("Angkatan harus berupa 4 digit tahun (maksimal %d)", currentYear))
	}

	if req.IPKTerakhir != nil {
		if *req.IPKTerakhir < 0.00 || *req.IPKTerakhir > 4.00 {
			errs["ipk_terakhir"] = append(errs["ipk_terakhir"], "IPK terakhir harus berada dalam rentang 0.00 - 4.00")
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateEnrollmentRequest memvalidasi input pengambilan mata kuliah (KRS).
func ValidateEnrollmentRequest(req model.CreateEnrollmentRequest) map[string][]string {
	errs := make(map[string][]string)

	if req.CourseID <= 0 {
		errs["course_id"] = append(errs["course_id"], "ID mata kuliah (course_id) wajib diisi dan harus valid")
	}

	if req.TahunAkademik == "" {
		errs["tahun_akademik"] = append(errs["tahun_akademik"], "Tahun akademik wajib diisi")
	} else if !tahunAkademikRegex.MatchString(req.TahunAkademik) {
		errs["tahun_akademik"] = append(errs["tahun_akademik"], "Format tahun akademik tidak valid (contoh format: 2026/2027-Ganjil)")
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
