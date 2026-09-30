package model

import "time"

// User merepresentasikan model pengguna di database.
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"` // 'admin' atau 'mahasiswa'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserInfo digunakan untuk data user dalam respons tanpa password.
type UserInfo struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// LoginRequest merepresentasikan request body untuk endpoint login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponseData merepresentasikan data yang dikembalikan pada saat login berhasil.
type LoginResponseData struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        UserInfo `json:"user"`
}

// AuthMeResponse merepresentasikan respons untuk endpoint GET /api/v1/auth/me.
type AuthMeResponse struct {
	ID      int            `json:"id"`
	Email   string         `json:"email"`
	Role    string         `json:"role"`
	Student *StudentProfile `json:"student,omitempty"`
}

// StudentProfile menyajikan ringkasan data mahasiswa untuk profil me.
type StudentProfile struct {
	ID          int      `json:"id"`
	NIM         string   `json:"nim"`
	Nama        string   `json:"nama"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir,omitempty"`
}
