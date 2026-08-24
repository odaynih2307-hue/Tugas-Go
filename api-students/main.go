package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/api/v1/students", getStudents)
	app.Get("/api/v1/students/:id", getStudent)
	app.Post("/api/v1/students", createStudent)
	app.Put("/api/v1/students/:id", updateStudent)
	app.Patch("/api/v1/students/:id", patchStudent)
	app.Delete("/api/v1/students/:id", deleteStudent)
	log.Fatal(app.Listen(":3000"))
}
