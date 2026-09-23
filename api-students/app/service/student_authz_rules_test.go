package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestCanAccessStudent(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin": {"student:list", "student:read:any", "student:create", "student:update:any", "student:delete"},
		"staff": {"student:list", "student:read:any", "student:create"},
		"user":  {},
	})

	ownerUser := model.AuthUser{UserID: 10, Username: "student_owner", Role: "user"}
	nonOwnerUser := model.AuthUser{UserID: 20, Username: "other_user", Role: "user"}
	staffUser := model.AuthUser{UserID: 30, Username: "staff_member", Role: "staff"}
	adminUser := model.AuthUser{UserID: 40, Username: "super_admin", Role: "admin"}

	const targetOwnerID = 10

	tests := []struct {
		name          string
		user          model.AuthUser
		ownerID       int
		anyPermission string
		expected      bool
	}{
		{
			name:          "Owner can read own student data",
			user:          ownerUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:read:any",
			expected:      true,
		},
		{
			name:          "Owner can update own student data",
			user:          ownerUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:update:any",
			expected:      true,
		},
		{
			name:          "Non-owner user cannot read other's student data",
			user:          nonOwnerUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:read:any",
			expected:      false,
		},
		{
			name:          "Non-owner user cannot update other's student data",
			user:          nonOwnerUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:update:any",
			expected:      false,
		},
		{
			name:          "Staff can read other's student data via student:read:any",
			user:          staffUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:read:any",
			expected:      true,
		},
		{
			name:          "Staff cannot update other's student data (no student:update:any)",
			user:          staffUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:update:any",
			expected:      false,
		},
		{
			name:          "Admin can read other's student data via student:read:any",
			user:          adminUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:read:any",
			expected:      true,
		},
		{
			name:          "Admin can update other's student data via student:update:any",
			user:          adminUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:update:any",
			expected:      true,
		},
		{
			name:          "Nil PermissionSet always fails closed",
			user:          nonOwnerUser,
			ownerID:       targetOwnerID,
			anyPermission: "student:read:any",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pSet := perms
			if tt.name == "Nil PermissionSet always fails closed" {
				pSet = nil
			}
			result := CanAccessStudent(tt.user, tt.ownerID, pSet, tt.anyPermission)
			if result != tt.expected {
				t.Errorf("CanAccessStudent(%+v, %d, ..., %q) = %v; want %v",
					tt.user, tt.ownerID, tt.anyPermission, result, tt.expected)
			}
		})
	}
}
