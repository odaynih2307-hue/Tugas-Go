package main

import "fmt"

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

// GetInfo menggunakan value receiver
// karena hanya membaca informasi Student.
func (s Student) GetInfo() string {
	return fmt.Sprintf(
		"ID: %d | Name: %s | Grade: %.2f | Active: %t",
		s.ID,
		s.Name,
		s.Grade,
		s.IsActive,
	)
}

// UpdateGrade menggunakan pointer receiver
// karena mengubah Grade pada Student.
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

// Activate menggunakan pointer receiver
// karena mengubah IsActive.
func (s *Student) Activate() {
	s.IsActive = true
}

// Deactivate menggunakan pointer receiver
// karena mengubah IsActive.
func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {
	student := Student{
		ID:       1,
		Name:     "Ody",
		Grade:    80,
		IsActive: false,
	}

	fmt.Println("=== DATA AWAL ===")
	fmt.Println(student.GetInfo())

	student.Activate()

	fmt.Println("\n=== SETELAH ACTIVATE ===")
	fmt.Println(student.GetInfo())

	student.UpdateGrade(92.5)

	fmt.Println("\n=== SETELAH UPDATE GRADE ===")
	fmt.Println(student.GetInfo())

	student.Deactivate()

	fmt.Println("\n=== SETELAH DEACTIVATE ===")
	fmt.Println(student.GetInfo())
}