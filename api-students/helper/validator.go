package helper

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate dibuat SEKALI untuk seluruh aplikasi.
//
// validator.New() melakukan refleksi dan menyimpan hasilnya dalam cache
// internal. Membuatnya ulang pada setiap request berarti membuang cache
// tersebut berkali-kali — mahal dan tidak ada gunanya.
var validate = newValidator()

var nimRegex = regexp.MustCompile(`^[0-9]{11,12}$`)

func newValidator() *validator.Validate {
	v := validator.New()

	// Tanpa ini, pesan error menyebut nama field Go ("Username"),
	// padahal client mengirim dan membaca nama JSON ("username").
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	// Aturan buatan sendiri. Aturan yang tidak disediakan library tetap
	// ditulis secara deklaratif sebagai tag, bukan dikembalikan menjadi
	// pemeriksaan manual yang tersebar di dalam service.
	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})

	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		str := fl.Field().String()
		if len(str) == 0 {
			return false
		}
		for _, r := range str {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
				r != '.' && r != '_' {
				return false
			}
		}
		return true
	})

	// Catatan Perbaikan Bug Modul Bagian B:
	// Pada modul Bagian B Langkah 5, baris ini tertulis:
	// `return passwordStrength(fl.Field().String()) != ""`
	// Padahal passwordStrength mengembalikan string kosong ("") jika password KUAT,
	// dan mengembalikan pesan error jika password LEMAH.
	// Logika != "" menyebabkan password kuat ditolak dan password lemah diterima.
	// Kami memperbaikinya menjadi: `== ""`
	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return passwordStrength(fl.Field().String()) == ""
	})

	// Custom validator untuk entity students (Tugas D.2):
	// Format NIM wajib 11-12 digit angka murni
	_ = v.RegisterValidation("validnim", func(fl validator.FieldLevel) bool {
		return nimRegex.MatchString(strings.TrimSpace(fl.Field().String()))
	})

	return v
}

// ValidateStruct menjalankan seluruh aturan pada tag struct dan
// mengembalikan peta nama field ke pesan berbahasa Indonesia.
// Mengembalikan nil berarti tidak ada pelanggaran.
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	// Terjadi bila yang dikirim bukan struct — itu kesalahan programmer,
	// bukan kesalahan pemakai API. Jangan diam-diam dianggap valid.
	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

// messageFor menerjemahkan nama tag menjadi kalimat yang dapat dibaca
// pemakai. Daftar ini terpusat: menambah satu tag baru cukup menambah
// satu case di sini, tidak menyebar ke banyak file.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "nilai maksimal " + fe.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "strongpassword":
		// Type assertion memakai bentuk DUA nilai, bukan satu. Bentuk
		// satu nilai akan panic bila suatu saat tag ini terpasang pada
		// field bukan string — mematikan server hanya karena salah tag.
		if value, ok := fe.Value().(string); ok {
			msg := passwordStrength(value)
			if msg != "" {
				return msg
			}
		}
		return "password tidak memenuhi syarat"
	case "validnim":
		return "format NIM tidak valid, harus 11-12 digit angka"
	case "oneof":
		return "harus salah satu dari: " +
			strings.ReplaceAll(fe.Param(), " ", ", ")
	default:
		// Jaring pengaman. Bila muncul di log, artinya ada tag yang
		// dipakai tetapi belum diterjemahkan di sini.
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

// passwordStrength memeriksa kelayakan password dan mengembalikan pesan kesalahan jika tidak memenuhi syarat.
// Mengembalikan string kosong "" jika password memenuhi seluruh kriteria keamanan.
func passwordStrength(p string) string {
	if len(p) < 8 {
		return "minimal 8 karakter"
	}

	// Pemeriksaan password umum
	lower := strings.ToLower(p)
	if lower == "password123" || lower == "12345678" || lower == "admin123" || lower == "qwerty123" {
		return "password terlalu umum"
	}

	hasLetter := false
	hasDigit := false
	for _, r := range p {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}

	return ""
}
