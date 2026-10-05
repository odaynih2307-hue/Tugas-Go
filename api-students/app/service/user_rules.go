package service

import (
	"strings"

	"api-students/app/model"
)

// ApplyUserPatch menggabungkan field PATCH yang ada ke model User saat ini.
func ApplyUserPatch(current model.User, req model.PatchUserRequest) model.User {
	if req.Username != nil {
		current.Username = strings.TrimSpace(*req.Username)
	}
	if req.Email != nil {
		current.Email = strings.TrimSpace(*req.Email)
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current
}

// IsEmptyUserPatch memeriksa body PATCH user yang tidak berisi field apa pun.
func IsEmptyUserPatch(req model.PatchUserRequest) bool {
	return req.Username == nil && req.Email == nil && req.IsActive == nil
}
