package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/service"
)

type CourseHandler struct {
	courseService service.CourseService
}

func NewCourseHandler(courseService service.CourseService) *CourseHandler {
	return &CourseHandler{courseService: courseService}
}

// List menangani GET /api/v1/courses (Semua role).
func (h *CourseHandler) List(c *fiber.Ctx) error {
	semester, _ := strconv.Atoi(c.Query("semester", "0"))
	available := strings.EqualFold(c.Query("available", "false"), "true")

	query := model.CourseQuery{
		Semester:  semester,
		Search:    c.Query("search", ""),
		Available: available,
	}

	courses, err := h.courseService.List(c.Context(), query)
	if err != nil {
		return helper.InternalError(c, err)
	}

	return helper.Success(c, fiber.StatusOK, "Daftar mata kuliah berhasil diambil", courses)
}
