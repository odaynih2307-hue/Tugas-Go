package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestCanAccessUser(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin": {"user:list", "user:read:any", "user:update:any", "user:delete", "role:assign"},
		"staff": {"user:list", "user:read:any"},
		"user":  {},
	})

	user1 := model.AuthUser{UserID: 1, Username: "budi", Role: "user"}
	user2 := model.AuthUser{UserID: 2, Username: "sari", Role: "user"}
	staff := model.AuthUser{UserID: 3, Username: "staff1", Role: "staff"}
	admin := model.AuthUser{UserID: 4, Username: "admin1", Role: "admin"}

	// Self access (Ownership)
	if !CanAccessUser(user1, user1.UserID, perms, "user:read:any") {
		t.Errorf("User should be able to access own data")
	}

	// Other user without permission
	if CanAccessUser(user1, user2.UserID, perms, "user:read:any") {
		t.Errorf("User should NOT be able to access other user's data")
	}

	// Staff with user:read:any
	if !CanAccessUser(staff, 1, perms, "user:read:any") {
		t.Errorf("Staff should be able to read any user data")
	}

	// Staff without user:update:any
	if CanAccessUser(staff, 1, perms, "user:update:any") {
		t.Errorf("Staff should NOT be able to update other user data")
	}

	// Admin with user:update:any
	if !CanAccessUser(admin, 1, perms, "user:update:any") {
		t.Errorf("Admin should be able to update any user data")
	}
}

func TestValidateAssignRole(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin": {},
		"staff": {},
		"user":  {},
	})

	admin := model.AuthUser{UserID: 1, Username: "admin", Role: "admin"}

	// Rule 1: Admin cannot change own role
	errs := ValidateAssignRole(admin, 1, model.AssignRoleRequest{Role: "staff"}, perms)
	if _, ok := errs["role"]; !ok {
		t.Errorf("Admin changing own role must fail validation")
	}

	// Rule 2: Unknown role must fail validation
	errs = ValidateAssignRole(admin, 2, model.AssignRoleRequest{Role: "hacker"}, perms)
	if _, ok := errs["role"]; !ok {
		t.Errorf("Assigning unknown role must fail validation")
	}

	// Rule 3: Empty role must fail validation
	errs = ValidateAssignRole(admin, 2, model.AssignRoleRequest{Role: ""}, perms)
	if _, ok := errs["role"]; !ok {
		t.Errorf("Assigning empty role must fail validation")
	}

	// Valid role assignment for other user
	errs = ValidateAssignRole(admin, 2, model.AssignRoleRequest{Role: "staff"}, perms)
	if len(errs) != 0 {
		t.Errorf("Assigning valid role to another user should have 0 errors, got: %v", errs)
	}
}
