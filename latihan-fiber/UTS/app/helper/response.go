package helper

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"uts-siakad/app/model"
	"uts-siakad/config"
)

// Success mengembalikan respons sukses seragam.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta mengembalikan respons sukses dengan objek pagination meta.
func SuccessWithMeta(c *fiber.Ctx, status int, message string, data any, meta *model.Meta) error {
	return c.Status(status).JSON(model.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// NoContent mengembalikan respons status 204 No Content tanpa payload.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Error mengembalikan respons error dengan pesan tertentu dan errors bernilai null.
func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.SimpleErrorResponse{
		Success: false,
		Message: message,
		Errors:  nil,
	})
}

// ValidationError mengembalikan respons 422 Unprocessable Entity dengan rincian per-field.
func ValidationError(c *fiber.Ctx, errors map[string][]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.ErrorResponse{
		Success: false,
		Message: "Validasi gagal",
		Errors:  errors,
	})
}

// InternalError mengembalikan respons 500 Internal Server Error tanpa mengekspos stack trace jika production.
func InternalError(c *fiber.Ctx, err error) error {
	log.Printf("[ERROR] Internal Server Error: %v", err)

	msg := "Terjadi kesalahan pada server"
	if config.AppConfig.AppEnv != "production" && err != nil {
		msg = err.Error()
	}

	return c.Status(fiber.StatusInternalServerError).JSON(model.SimpleErrorResponse{
		Success: false,
		Message: msg,
		Errors:  nil,
	})
}
