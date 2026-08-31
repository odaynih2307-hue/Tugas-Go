package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// Response berhasil dengan status 200.
func OK(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Response berhasil untuk daftar data dengan informasi pagination.
func OKList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Response berhasil membuat data baru dengan status 201 dan header Location.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)

	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Response berhasil tanpa body dengan status 204.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Response error umum.
func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}

// Response khusus validasi dengan status 422.
func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false,
		Message: "validasi gagal",
		Errors:  errs,
	})
}
