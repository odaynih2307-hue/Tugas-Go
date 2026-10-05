package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthUser menyimpan data identitas user yang sedang terautentikasi pada request context.
type AuthUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// AssignRoleRequest dipakai endpoint PATCH /users/:id/role.
type AssignRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin staff user"`
}

// Mulai pertemuan ini, aturan validasi ditulis sebagai tag pada struct.
// Aturan dan bentuk data berada pada baris yang sama, sehingga menambah
// satu field tanpa aturannya menjadi kelalaian yang langsung terlihat.
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,min=8,max=72,nospace"`
}

type ReplaceUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" validate:"required,email,max=120"`
	IsActive bool   `json:"is_active"`
}

// Pada PATCH, pointer membedakan "tidak dikirim" (nil) dari "dikirim
// bernilai kosong". omitnil dipilih karena ia menyatakan maksud yang
// sebenarnya: lewati hanya bila nil.
//
// Catatan Perbaikan Bug Modul Bagian B:
// Pada Bagian B Langkah 6, field Username bertipe `string` (bukan pointer `*string`).
// Hal ini menyebabkan compiler menolak perbandingan `req.Username != nil` dan `*req.Username`.
// Kami memperbaikinya dengan mengubah tipe Username menjadi `*string`.
type PatchUserRequest struct {
	Username *string `json:"username,omitempty" validate:"omitnil,min=3,max=30,alphanum"`
	Email    *string `json:"email,omitempty" validate:"omitnil,email,max=120"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,username"`
	Email    string `json:"email" validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,max=72,strongpassword"`
}

// ErrorResponse adalah bentuk seragam untuk seluruh response kegagalan API.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// Cursor penanda posisi keyset pagination
type Cursor struct {
	CreatedAt time.Time
	ID        int
}

// CursorMeta menggantikan Meta pada endpoint yang memakai cursor.
//
// Perhatikan tidak adanya Total dan TotalPages. Keduanya tidak dapat
// disediakan tanpa COUNT(*) atas seluruh tabel — persis biaya yang ingin
// dihindari oleh pagination berbasis cursor.
type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

type CursorQuery struct {
	Limit    int
	After    *Cursor
	Search   string
	IsActive *bool
}
