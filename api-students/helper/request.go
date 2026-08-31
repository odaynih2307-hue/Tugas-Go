package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// ReqCtx memberi batas waktu context 5 detik untuk operasi basis data.
func ReqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID membaca ID dari URL params dan memvalidasi nilainya berupa angka positif.
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

// allowedSort mendefinisikan kolom yang diizinkan untuk diurutkan.
var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

// ParseListQuery membaca query string dengan nilai default yang aman.
func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}

	if q.Page < 1 {
		q.Page = 1
	}

	if q.Limit < 1 {
		q.Limit = 10
	}

	if q.Limit > 100 {
		q.Limit = 100
	}

	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}

	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	return q
}
