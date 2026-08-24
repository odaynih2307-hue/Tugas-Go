package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

var students = []Student{
	{
		ID:        1,
		NIM:       "082211133001",
		Name:      "Andi",
		Grade:     85,
		IsActive:  true,
		CreatedAt: time.Now(),
	},
	{
		ID:        2,
		NIM:       "082211133002",
		Name:      "Budi",
		Grade:     78,
		IsActive:  true,
		CreatedAt: time.Now(),
	},
	{
		ID:        3,
		NIM:       "082211133003",
		Name:      "Sari",
		Grade:     92,
		IsActive:  false,
		CreatedAt: time.Now(),
	},
}

func getStudents(c *fiber.Ctx) error {
	query := parseListQuery(c)

	data := students

	// SEARCH
	if query.Search != "" {
		search := strings.ToLower(query.Search)

		filtered := make([]Student, 0)

		for _, student := range data {
			if strings.Contains(strings.ToLower(student.Name), search) {
				filtered = append(filtered, student)
			}
		}

		data = filtered
	}

	// FILTER IS_ACTIVE
	if c.Context().QueryArgs().Has("is_active") {
		isActive := c.Query("is_active")

		filtered := make([]Student, 0)

		for _, student := range data {
			if isActive == "true" && student.IsActive {
				filtered = append(filtered, student)
			}

			if isActive == "false" && !student.IsActive {
				filtered = append(filtered, student)
			}
		}

		data = filtered
	}

	// SORTING
	if query.Sort != "" {
		order := strings.ToLower(query.Order)

		sort.Slice(data, func(i, j int) bool {
			switch query.Sort {
			case "name":
				if order == "desc" {
					return data[i].Name > data[j].Name
				}
				return data[i].Name < data[j].Name

			case "grade":
				if order == "desc" {
					return data[i].Grade > data[j].Grade
				}
				return data[i].Grade < data[j].Grade

			case "nim":
				if order == "desc" {
					return data[i].NIM > data[j].NIM
				}
				return data[i].NIM < data[j].NIM

			case "created_at":
				if order == "desc" {
					return data[i].CreatedAt.After(data[j].CreatedAt)
				}
				return data[i].CreatedAt.Before(data[j].CreatedAt)

			default:
				return false
			}
		})
	}

	// VALIDASI PAGE
	if query.Page < 1 {
		query.Page = 1
	}

	// VALIDASI LIMIT
	if query.Limit <= 0 {
		query.Limit = 10
	}

	if query.Limit > 100 {
		query.Limit = 100
	}

	// TOTAL DATA SETELAH SEARCH
	total := len(data)

	// TOTAL HALAMAN
	totalPages := 0

	if total > 0 {
		totalPages = (total + query.Limit - 1) / query.Limit
	}

	// JIKA PAGE MELEBIHI TOTAL PAGE
	if totalPages > 0 && query.Page > totalPages {
		query.Page = totalPages
	}

	// PAGINATION
	start := (query.Page - 1) * query.Limit
	end := start + query.Limit

	if start > total {
		start = total
	}

	if end > total {
		end = total
	}

	data = data[start:end]

	return okList(
		c,
		"daftar student berhasil diambil",
		data,
		&Meta{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	)
}
func getStudent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	for _, student := range students {
		if student.ID == id {
			return ok(c, "student ditemukan", student)
		}
	}

	return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
}

func createStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	errs := make(map[string]string)

	if req.NIM == "" {
		errs["nim"] = "NIM wajib diisi"
	}

	if req.Name == "" {
		errs["name"] = "nama wajib diisi"
	}

	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	for _, student := range students {
		if student.NIM == req.NIM {
			return fail(c, fiber.StatusConflict, "NIM sudah digunakan")
		}
	}

	newID := 1

	if len(students) > 0 {
		newID = students[len(students)-1].ID + 1
	}

	newStudent := Student{
		ID:        newID,
		NIM:       req.NIM,
		Name:      req.Name,
		Grade:     req.Grade,
		IsActive:  req.IsActive,
		CreatedAt: time.Now(),
	}

	students = append(students, newStudent)

	return created(
		c,
		"student berhasil dibuat",
		newStudent,
		"/api/v1/students/"+strconv.Itoa(newStudent.ID),
	)
}

func updateStudent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req ReplaceStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	errs := make(map[string]string)

	if req.NIM == "" {
		errs["nim"] = "NIM wajib diisi"
	}

	if req.Name == "" {
		errs["name"] = "nama wajib diisi"
	}

	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "grade harus antara 0 dan 100"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	// Cari student berdasarkan ID.
	index := -1

	for i, student := range students {
		if student.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	// Cek NIM agar tidak bentrok dengan student lain.
	for i, student := range students {
		if i != index && student.NIM == req.NIM {
			return fail(c, fiber.StatusConflict, "NIM sudah digunakan")
		}
	}

	// PUT mengganti seluruh data student.
	students[index].NIM = req.NIM
	students[index].Name = req.Name
	students[index].Grade = req.Grade
	students[index].IsActive = req.IsActive

	return ok(
		c,
		"student berhasil diperbarui",
		students[index],
	)
}

func patchStudent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	index := -1

	for i, student := range students {
		if student.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	var req PatchStudentRequest

	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	if req.NIM == nil &&
		req.Name == nil &&
		req.Grade == nil &&
		req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	if req.NIM != nil {
		if *req.NIM == "" {
			return failValidation(c, map[string]string{
				"nim": "NIM tidak boleh kosong",
			})
		}

		for i, student := range students {
			if i != index && student.NIM == *req.NIM {
				return fail(c, fiber.StatusConflict, "NIM sudah digunakan")
			}
		}

		students[index].NIM = *req.NIM
	}

	if req.Name != nil {
		if *req.Name == "" {
			return failValidation(c, map[string]string{
				"name": "nama tidak boleh kosong",
			})
		}

		students[index].Name = *req.Name
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return failValidation(c, map[string]string{
				"grade": "grade harus antara 0 dan 100",
			})
		}

		students[index].Grade = *req.Grade
	}

	if req.IsActive != nil {
		students[index].IsActive = *req.IsActive
	}

	return ok(
		c,
		"student berhasil diperbarui sebagian",
		students[index],
	)
}

func deleteStudent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	index := -1

	for i, student := range students {
		if student.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	students = append(students[:index], students[index+1:]...)

	return c.SendStatus(fiber.StatusNoContent)
}
