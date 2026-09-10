package helper

import "testing"

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{
			name:     "password kuat",
			password: "Password123!",
			expected: true,
		},
		{
			name:     "password terlalu pendek",
			password: "Pa1!",
			expected: false,
		},
		{
			name:     "password tanpa huruf besar",
			password: "password123!",
			expected: false,
		},
		{
			name:     "password tanpa angka",
			password: "Password!",
			expected: false,
		},
		{
			name:     "password tanpa simbol",
			password: "Password123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStrongPassword(tt.password)

			if result != tt.expected {
				t.Errorf(
					"IsStrongPassword(%q) = %v, expected %v",
					tt.password,
					result,
					tt.expected,
				)
			}
		})
	}
}
