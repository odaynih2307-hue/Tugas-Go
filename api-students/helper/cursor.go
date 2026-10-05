package helper

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// EncodeCursor mengubah penanda menjadi satu string yang aman di URL.
//
// base64 dipakai agar bentuk internalnya dapat kita ubah tanpa mengubah
// kontrak API. Perlu ditegaskan: base64 adalah PENGKODEAN, bukan
// enkripsi. Siapa pun dapat membacanya, dan siapa pun dapat menyusun
// cursor palsu. Karena itu cursor tidak boleh memuat apa pun yang
// bersifat rahasia atau yang dipercaya sebagai dasar hak akses.
func EncodeCursor(createdAt time.Time, id int) string {
	raw := strconv.FormatInt(createdAt.UTC().UnixNano(), 10) +
		"|" + strconv.Itoa(id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor membaca kembali penanda dari string.
//
// Seluruh jalur kegagalan mengembalikan error, tidak ada yang "diperbaiki
// diam-diam". Cursor rusak berarti permintaan client memang salah, dan
// client berhak tahu itu lewat 400 — bukan diam-diam dikembalikan ke
// halaman pertama seolah tidak terjadi apa-apa.
func DecodeCursor(encoded string) (model.Cursor, error) {
	if strings.TrimSpace(encoded) == "" {
		return model.Cursor{}, ErrInvalidCursor
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return model.Cursor{}, ErrInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id < 1 {
		return model.Cursor{}, ErrInvalidCursor
	}
	return model.Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: id}, nil
}

// ParseCursorQuery membaca parameter query keyset pagination: limit, cursor, search, is_active.
func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	q := model.CursorQuery{
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		} else {
			return q, BadRequest("parameter is_active harus berupa boolean")
		}
	}

	cursorStr := strings.TrimSpace(c.Query("cursor"))
	if cursorStr != "" {
		cur, err := DecodeCursor(cursorStr)
		if err != nil {
			return q, BadRequest("cursor tidak valid")
		}
		q.After = &cur
	}

	return q, nil
}
