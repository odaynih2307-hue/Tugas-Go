package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

func TestValidateCreateStudent(t *testing.T) {
	tests := []struct {
		name    string
		input   model.CreateStudentRequest
		wantErr bool
	}{
		{
			name: "data valid dengan NIM 12 digit",
			input: model.CreateStudentRequest{
				NIM:   "082211133002",
				Name:  "Budi Santoso",
				Grade: 85,
			},
			wantErr: false,
		},
		{
			name: "NIM kosong",
			input: model.CreateStudentRequest{
				NIM:   "",
				Name:  "Budi Santoso",
				Grade: 85,
			},
			wantErr: true,
		},
		{
			name: "NIM format salah (kurang dari 11 digit)",
			input: model.CreateStudentRequest{
				NIM:   "12345",
				Name:  "Budi Santoso",
				Grade: 85,
			},
			wantErr: true,
		},
		{
			name: "nama terlalu pendek (< 3 karakter)",
			input: model.CreateStudentRequest{
				NIM:   "082211133002",
				Name:  "Al",
				Grade: 85,
			},
			wantErr: true,
		},
		{
			name: "grade lebih dari 100",
			input: model.CreateStudentRequest{
				NIM:   "082211133002",
				Name:  "Budi Santoso",
				Grade: 120,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := helper.ValidateStruct(tt.input)

			if (len(errs) > 0) != tt.wantErr {
				t.Errorf(
					"ValidateStruct() errors = %v, wantErr %v",
					errs,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateReplaceStudent(t *testing.T) {
	tests := []struct {
		name    string
		input   model.ReplaceStudentRequest
		wantErr bool
	}{
		{
			name: "data valid",
			input: model.ReplaceStudentRequest{
				NIM:      "082211133002",
				Name:     "Budi Santoso",
				Grade:    90,
				IsActive: true,
			},
			wantErr: false,
		},
		{
			name: "nama kosong",
			input: model.ReplaceStudentRequest{
				NIM:      "082211133002",
				Name:     "",
				Grade:    90,
				IsActive: true,
			},
			wantErr: true,
		},
		{
			name: "grade negatif",
			input: model.ReplaceStudentRequest{
				NIM:      "082211133002",
				Name:     "Budi Santoso",
				Grade:    -10,
				IsActive: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := helper.ValidateStruct(tt.input)

			if (len(errs) > 0) != tt.wantErr {
				t.Errorf(
					"ValidateStruct() errors = %v, wantErr %v",
					errs,
					tt.wantErr,
				)
			}
		})
	}
}

func TestApplyPatchStudent(t *testing.T) {
	nameBaru := "Andi Wijaya"
	gradeBaru := 95.0

	current := model.Student{
		ID:       1,
		NIM:      "082211133001",
		Name:     "Budi Santoso",
		Grade:    80,
		IsActive: true,
	}

	req := model.PatchStudentRequest{
		Name:  &nameBaru,
		Grade: &gradeBaru,
	}

	// Validasi deklaratif menggunakan omitnil
	errs := helper.ValidateStruct(req)
	if len(errs) > 0 {
		t.Fatalf("ValidateStruct() unexpected error: %v", errs)
	}

	result := ApplyPatchStudent(current, req)

	if result.Name != "Andi Wijaya" {
		t.Errorf(
			"Name = %s, ingin %s",
			result.Name,
			"Andi Wijaya",
		)
	}

	if result.Grade != 95 {
		t.Errorf(
			"Grade = %v, ingin 95",
			result.Grade,
		)
	}

	// Field yang tidak dikirim tidak boleh berubah.
	if result.NIM != current.NIM {
		t.Errorf(
			"NIM berubah menjadi %s, seharusnya %s",
			result.NIM,
			current.NIM,
		)
	}

	if result.IsActive != current.IsActive {
		t.Errorf(
			"IsActive berubah",
		)
	}
}
