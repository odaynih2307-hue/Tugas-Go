package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Role      string    `json:"role"`
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
	Role string `json:"role"`
}

// ErrorResponse adalah bentuk seragam untuk seluruh response kegagalan API.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}
