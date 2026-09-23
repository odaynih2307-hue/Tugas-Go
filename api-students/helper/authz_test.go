package helper_test

import (
	"reflect"
	"testing"

	"api-students/helper"
)

func TestPermissionSet_Can(t *testing.T) {
	raw := map[string][]string{
		"admin": {"user:list", "user:read:any", "user:update:any", "user:delete", "role:assign", "student:list"},
		"staff": {"user:list", "user:read:any", "student:list"},
		"user":  {},
	}

	p := helper.NewPermissionSet(raw)

	tests := []struct {
		name       string
		pSet       *helper.PermissionSet
		role       string
		permission string
		expected   bool
	}{
		{"Admin has user:delete", p, "admin", "user:delete", true},
		{"Admin has role:assign", p, "admin", "role:assign", true},
		{"Staff has user:list", p, "staff", "user:list", true},
		{"Staff does NOT have user:delete", p, "staff", "user:delete", false},
		{"User has NO permissions", p, "user", "user:list", false},
		{"Unknown role returns false (Fail Closed)", p, "guest", "user:list", false},
		{"Unknown permission returns false (Fail Closed)", p, "admin", "system:nuke", false},
		{"Nil PermissionSet returns false (Fail Closed)", nil, "admin", "user:list", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pSet.Can(tt.role, tt.permission)
			if got != tt.expected {
				t.Errorf("Can(%q, %q) = %v; want %v", tt.role, tt.permission, got, tt.expected)
			}
		})
	}
}

func TestPermissionSet_KnownRoles(t *testing.T) {
	raw := map[string][]string{
		"user":  {},
		"admin": {"user:list"},
		"staff": {"user:list"},
	}

	p := helper.NewPermissionSet(raw)
	roles := p.KnownRoles()

	expected := []string{"admin", "staff", "user"}
	if !reflect.DeepEqual(roles, expected) {
		t.Errorf("KnownRoles() = %v; want %v", roles, expected)
	}

	if !p.IsKnownRole("admin") || !p.IsKnownRole("staff") || !p.IsKnownRole("user") {
		t.Errorf("IsKnownRole failed for valid roles")
	}

	if p.IsKnownRole("superadmin") {
		t.Errorf("IsKnownRole returned true for unknown role")
	}

	var nilP *helper.PermissionSet
	if nilP.IsKnownRole("admin") {
		t.Errorf("nil PermissionSet IsKnownRole must return false")
	}
	if len(nilP.KnownRoles()) != 0 {
		t.Errorf("nil PermissionSet KnownRoles must return empty slice")
	}
}
