package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// OK mengembalikan response berhasil standar dengan status 200.
func OK(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Success mengembalikan response berhasil dengan status HTTP custom.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessCursor mengembalikan response berhasil untuk endpoint yang memakai cursor pagination.
func SuccessCursor(c *fiber.Ctx, message string, data any, meta *model.CursorMeta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebCursorResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// OKList mengembalikan response berhasil untuk daftar data dengan informasi pagination offset.
func OKList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengembalikan response berhasil membuat data baru dengan status 201 dan header Location.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)

	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengembalikan response berhasil tanpa body dengan status 204.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// CATATAN ARSITEKTUR MODUL 7:
// helper.Fail dan helper.FailValidation DIHAPUS seluruhnya sesuai instruksi Langkah 4 modul:
// "Hapus helper.Fail dan helper.FailValidation seluruhnya. Selama keduanya masih ada,
// akan selalu ada godaan memakainya — dan satu pemakaian saja sudah cukup untuk membuat
// bentuk response tidak lagi seragam. Menghapusnya membuat compiler yang menegakkan aturan ini."
