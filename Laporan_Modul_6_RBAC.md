# LAPORAN PRAKTIKUM PEMROGRAMAN BACKEND LANJUT
## MODUL 6: AUTHORIZATION & ROLE BASED ACCESS CONTROL (RBAC)

**Program Studi**: D4 Teknik Informatika — Fakultas Vokasi, Universitas Airlangga  
**Mata Kuliah**: Praktikum Pemrograman Backend Lanjut (2 SKS)  
**Tautan Repository**: [https://github.com/odaynih2307-hue/Tugas-Go.git](https://github.com/odaynih2307-hue/Tugas-Go.git)  

---

## DAFTAR ISI
1. [Ringkasan Eksekutif & Landasan Teori](#1-ringkasan-eksekutif--landasan-teori)
2. [Langkah Pengerjaan & Potongan Kode Penting](#2-langkah-pengerjaan--potongan-kode-penting)
   - 2.1 Migrasi Skema Database RBAC & Students (`003_rbac.sql` & `004_student_permissions.sql`)
   - 2.2 Struktur In-Memory `PermissionSet` (`helper/authz.go`)
   - 2.3 Repository Layer (`role_repository.go`, `user_repository.go`, `student_repository.go`)
   - 2.4 Middleware Layer (`middleware/authz.go` & `middleware/request_logger.go`)
   - 2.5 Aturan Otorisasi Murni (`authz_rules.go` & `student_authz_rules.go`)
   - 2.6 Service Layer (`user_service.go`, `student_service.go`, `auth_service.go`)
   - 2.7 Peta Hak Akses Terpusat (`route/route.go`)
3. [Tabel Matriks Hak Akses & Bukti Pengujian Aktual](#3-tabel-matriks-hak-akses--bukti-pengujian-aktual)
   - 3.1 Matriks Hak Akses Endpoint Users
   - 3.2 Matriks Hak Akses Endpoint Students
4. [Bukti Pengujian Negatif & Kasus Khusus](#4-bukti-pengujian-negatif--kasus-khusus)
5. [Jawaban Analisis Singkat C.4](#5-jawaban-analisis-singkat-c4)
6. [Jawaban Eksplorasi Mandiri](#6-jawaban-eksplorasi-mandiri)
7. [Kesimpulan](#7-kesimpulan)

---

## 1. RINGKASAN EKSEKUTIF & LANDASAN TEORI

Pada Modul 5, sistem autentikasi berbasis JWT (*JSON Web Token*) telah berhasil diimplementasikan dengan menyisipkan atribut `role` ke dalam claims token. Namun, autentikasi saja hanya menjawab pertanyaan **"Siapa Anda?"** (*Authentication*), belum menjawab pertanyaan **"Anda berhak melakukan apa?"** (*Authorization*). Tanpa lapisan otorisasi, siapa pun yang berhasil mendaftar dan login (bahkan user biasa) dapat membaca seluruh database user, mengubah data orang lain, ataupun menghapusnya.

Modul 6 mengimplementasikan sistem otorisasi tingkat lanjut dengan menggabungkan dua paradigma utama:
1. **Role Based Access Control (RBAC)**: Pengguna dikaitkan dengan satu *Role*, dan setiap *Role* memiliki sekumpulan *Permission* yang tersimpan di basis data relasional. Keputusan akses yang independen terhadap isi data (seperti `GET /users`, `POST /students`, `DELETE /students/:id`) dievaluasi secara deklaratif di lapisan Middleware (`RequirePermission`).
2. **Ownership-Based Access Control (ABAC Subset)**: Keputusan akses yang bergantung pada konteks kepemilikan baris data (seperti `GET /students/:id`, `PUT /students/:id`, `PATCH /students/:id`) dievaluasi di lapisan Service menggunakan fungsi murni `CanAccessStudent`. Pemilik data (*owner*) selalu diizinkan mengelola datanya sendiri, sedangkan pihak lain diwajibkan memiliki permission khusus bertingkat *any* (`student:read:any`, `student:update:any`).
3. **Prinsip Keamanan Kunci**:
   - **Fail Closed**: Segala bentuk kegagalan (role tidak dikenal, permission tidak terdaftar, pointer bernilai `nil`, atau parameter tidak sah) secara mutlak menolak akses (mengembalikan `false` atau status code `403 Forbidden` / `401 Unauthorized`).
   - **Mitigasi Timing Attack**: Pemeriksaan otorisasi dilakukan *sebelum* eksekusi query data (atau dengan waktu tanggap seragam) agar penyerang tidak dapat melakukan *ID Enumeration* dari perbedaan latensi antara `403` dan `404`.
   - **Stateless JWT Trade-off**: Perubahan role di database tidak seketika mengubah hak pada token aktif yang belum kadaluarsa; hal ini dimitigasi dengan masa berlaku access token yang singkat (15 menit).

---

## 2. LANGKAH PENGERJAAN & POTONGAN KODE PENTING

### 2.1 Migrasi Skema Database RBAC & Students

#### `migrations/003_rbac.sql`
Skema relasional RBAC terdiri dari tabel `roles`, `permissions`, dan tabel pivot `role_permissions` dengan composite primary key.
```sql
-- roles: daftar role yang diakui sistem
CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'Akses penuh terhadap seluruh data dan pengaturan'),
    ('staff', 'Boleh melihat data seluruh user, tetapi tidak boleh mengubah'),
    ('user', 'Hanya boleh mengelola datanya sendiri')
ON CONFLICT (name) DO NOTHING;

-- permissions: daftar tindakan yang dapat diberikan kepada role
CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

INSERT INTO permissions (name, description) VALUES
    ('user:list', 'Melihat daftar seluruh user'),
    ('user:read:any', 'Melihat data user mana pun'),
    ('user:update:any', 'Mengubah data user mana pun'),
    ('user:delete', 'Menghapus user'),
    ('role:assign', 'Mengubah role milik user lain')
ON CONFLICT (name) DO NOTHING;

-- role_permissions: tabel penghubung inti RBAC
CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(20) NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'user:list'),
    ('admin', 'user:read:any'),
    ('admin', 'user:update:any'),
    ('admin', 'user:delete'),
    ('admin', 'role:assign'),
    ('staff', 'user:list'),
    ('staff', 'user:read:any')
ON CONFLICT DO NOTHING;

-- Relasi dan indeks ke tabel users
UPDATE users SET role = 'user' WHERE role IS NULL OR role NOT IN (SELECT name FROM roles);
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_fkey;
ALTER TABLE users ADD CONSTRAINT users_role_fkey FOREIGN KEY (role) REFERENCES roles(name) ON UPDATE CASCADE;
CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
```

#### `migrations/004_student_permissions.sql`
Menambahkan permission untuk entitas mahasiswa serta kolom `owner_id` yang terikat relasi Foreign Key ke tabel `users`.
```sql
INSERT INTO permissions (name, description) VALUES
    ('student:list', 'Melihat daftar mahasiswa'),
    ('student:read:any', 'Melihat data mahasiswa milik siapa pun'),
    ('student:create', 'Menambahkan data mahasiswa baru'),
    ('student:update:any', 'Mengubah data mahasiswa milik siapa pun'),
    ('student:delete', 'Menghapus data mahasiswa')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- Penambahan owner_id dengan migrasi aman untuk baris lama
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

UPDATE students 
SET owner_id = (SELECT id FROM users ORDER BY id ASC LIMIT 1)
WHERE owner_id IS NULL AND EXISTS (SELECT 1 FROM users);

ALTER TABLE students DROP CONSTRAINT IF EXISTS fk_students_owner;
ALTER TABLE students ADD CONSTRAINT fk_students_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS students_owner_idx ON students (owner_id);
```

---

### 2.2 Struktur In-Memory `PermissionSet` (`helper/authz.go`)

Untuk menghindari query basis data berulang pada setiap request HTTP, seluruh pemetaan hak akses dimuat sekali saat startup ke dalam memori dengan representasi map bersarang `map[string]map[string]struct{}` untuk pencarian $O(1)$.

```go
package helper

import "sort"

type PermissionSet struct {
	byRole map[string]map[string]struct{}
}

func NewPermissionSet(raw map[string][]string) *PermissionSet {
	byRole := make(map[string]map[string]struct{}, len(raw))
	for role, permissions := range raw {
		set := make(map[string]struct{}, len(permissions))
		for _, permission := range permissions {
			set[permission] = struct{}{}
		}
		byRole[role] = set
	}
	return &PermissionSet{byRole: byRole}
}

// Can mengevaluasi apakah role memiliki permission dengan prinsip fail-closed.
func (p *PermissionSet) Can(role, permission string) bool {
	if p == nil {
		return false
	}
	permissions, ok := p.byRole[role]
	if !ok {
		return false
	}
	_, granted := permissions[permission]
	return granted
}

func (p *PermissionSet) PermissionsOf(role string) []string {
	result := []string{}
	if p == nil {
		return result
	}
	for permission := range p.byRole[role] {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func (p *PermissionSet) KnownRoles() []string {
	result := []string{}
	if p == nil {
		return result
	}
	for role := range p.byRole {
		result = append(result, role)
	}
	sort.Strings(result)
	return result
}

func (p *PermissionSet) IsKnownRole(role string) bool {
	if p == nil {
		return false
	}
	_, ok := p.byRole[role]
	return ok
}
```

---

### 2.3 Repository Layer

#### `app/repository/role_repository.go`
Mengambil data pemetaan menggunakan `LEFT JOIN` dan `COALESCE` sehingga role yang belum memiliki permission (seperti `user`) tetap terdaftar.
```go
package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepository interface {
	LoadPermissions(ctx context.Context) (map[string][]string, error)
}

type rolePostgresRepository struct{ pool *pgxpool.Pool }

func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &rolePostgresRepository{pool: pool}
}

func (r *rolePostgresRepository) LoadPermissions(ctx context.Context) (map[string][]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.name, COALESCE(rp.permission_name, '')
		 FROM roles r
		 LEFT JOIN role_permissions rp ON rp.role_name = r.name
		 ORDER BY r.name, rp.permission_name`)
	if err != nil {
		return nil, fmt.Errorf("mengambil permission: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(&role, &permission); err != nil {
			return nil, fmt.Errorf("membaca row permission: %w", err)
		}
		if _, ok := result[role]; !ok {
			result[role] = []string{}
		}
		if permission != "" {
			result[role] = append(result[role], permission)
		}
	}
	return result, rows.Err()
}
```

#### `app/repository/user_repository.go` — Method `UpdateRole`
```go
func (r *UserRepository) UpdateRole(ctx context.Context, id int, role string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET role = $1
		 WHERE id = $2
		 RETURNING id, username, password, role, created_at`,
		role, id,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("mengubah role user: %w", err)
	}
	return &user, nil
}
```

#### `app/repository/student_repository.go` — `Create` & `Update` dengan `owner_id`
```go
func (r *studentPostgresRepository) Create(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}
	return s, nil
}
```

---

### 2.4 Middleware Layer

#### `middleware/authz.go`
```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"api-students/helper"
)

func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}
		if !perms.Can(user.Role, permission) {
			return helper.Fail(c, fiber.StatusForbidden,
				"role "+user.Role+" tidak memiliki hak "+permission)
		}
		return c.Next()
	}
}
```

#### `middleware/request_logger.go`
Mencatat jejak audit termasuk `user_id` dan `role` pada setiap request yang terautentikasi.
```go
package middleware

import (
	"log/slog"
	"time"
	"github.com/gofiber/fiber/v2"
	"api-students/helper"
)

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		attrs := []any{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		if user, ok := helper.CurrentUser(c); ok {
			attrs = append(attrs,
				slog.Int("user_id", user.UserID),
				slog.String("role", user.Role),
			)
		}

		logger.Info("http_request", attrs...)
		return err
	}
}
```

---

### 2.5 Aturan Otorisasi Murni

#### `app/service/student_authz_rules.go`
Fungsi murni tanpa ketergantungan framework untuk memeriksa kepemilikan (*ownership*) dan permission peran:
```go
package service

import (
	"api-students/app/model"
	"api-students/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// Jalur 1: Kepemilikan (Ownership)
	if ownerID > 0 && current.UserID == ownerID {
		return true
	}

	// Jalur 2: Hak peran global (:any)
	return perms.Can(current.Role, anyPermission)
}
```

#### `app/service/authz_rules.go`
```go
package service

import (
	"strings"
	"api-students/app/model"
	"api-students/helper"
)

func CanAccessUser(
	current model.AuthUser,
	targetID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == targetID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}

func ValidateAssignRole(
	current model.AuthUser,
	targetID int,
	req model.AssignRoleRequest,
	perms *helper.PermissionSet,
) map[string]string {
	errs := map[string]string{}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		errs["role"] = "wajib diisi"
		return errs
	}
	if !perms.IsKnownRole(role) {
		errs["role"] = "role tidak dikenal, pilih salah satu dari: " +
			strings.Join(perms.KnownRoles(), ", ")
	}
	if current.UserID == targetID {
		errs["role"] = "tidak boleh mengubah role diri sendiri"
	}
	return errs
}
```

---

### 2.6 Service Layer

#### `app/service/student_service.go`
Pemeriksaan hak kepemilikan dan pengisian `owner_id` otomatis dari token pemanggil:
```go
// POST /students -> owner_id diisi dari identitas pemanggil
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "format JSON tidak valid")
	}

	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// owner_id dikunci dari current.UserID
	baru, err := s.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: isActive,
		OwnerID:  current.UserID,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal menyimpan student")
	}
	return helper.Created(c, "student berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

// GET /students/:id -> Pemeriksaan kepemilikan
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id tidak valid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data student")
	}

	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengakses data student ini")
	}

	return helper.OK(c, "student ditemukan", student)
}
```

---

### 2.7 Peta Hak Akses Terpusat (`route/route.go`)

```go
package route

import (
	"github.com/gofiber/fiber/v2"
	"api-students/helper"
	"api-students/middleware"
)

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- Publik ---
	api.Get("/health", healthCheck(deps.Pool))

	// --- Autentikasi ---
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	perms := deps.Permissions

	// --- Users Endpoint ---
	users := api.Group("/users", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	users.Get("/", middleware.RequirePermission(perms, "user:list"), deps.UserService.List)
	users.Post("/", middleware.RequirePermission(perms, "user:update:any"), deps.UserService.Create)
	users.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), deps.UserService.Delete)
	users.Patch("/:id/role", middleware.RequirePermission(perms, "role:assign"), deps.UserService.AssignRole)
	users.Get("/:id", deps.UserService.Get)
	users.Put("/:id", deps.UserService.Replace)
	users.Patch("/:id", deps.UserService.Patch)

	// --- Students Endpoint ---
	students := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Post("/", middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
}
```

---

## 3. TABEL MATRIKS HAK AKSES & BUKTI PENGUJIAN AKTUAL

Pengujian dilakukan secara otomatis menggunakan automated test runner terhadap server HTTP yang aktif pada `http://localhost:3000/api/v1`. Seluruh status code di bawah ini merupakan **hasil pengamatan langsung (aktual)**.

### 3.1 Matriks Hak Akses Endpoint Users (Langkah 9)

| Endpoint & Skenario | Method | `admin` | `staff` | `user` | Tanpa Auth | Status Verifikasi |
|---|---|:---:|:---:|:---:|:---:|:---:|
| `GET /users` (Daftar semua user) | GET | **200** | **200** | **403** | **401** | ✅ Verified |
| `GET /users/:id` (Data diri sendiri) | GET | **200** | **200** | **200** | **401** | ✅ Verified |
| `GET /users/:id` (Data orang lain) | GET | **200** | **200** | **403** | **401** | ✅ Verified |
| `PUT /users/:id` (Edit data orang lain) | PUT | **200** | **403** | **403** | **401** | ✅ Verified |
| `DELETE /users/:id` (Hapus user lain) | DELETE | **204** | **403** | **403** | **401** | ✅ Verified |
| `DELETE /users/:id` (Hapus akun sendiri) | DELETE | **403** | **403** | **403** | **401** | ✅ Verified |
| `PATCH /users/:id/role` (Ubah role user lain) | PATCH | **200** | **403** | **403** | **401** | ✅ Verified |
| `PATCH /users/:id/role` (Ubah role diri sendiri) | PATCH | **422** | **403** | **403** | **401** | ✅ Verified |

---

### 3.2 Matriks Hak Akses Endpoint Students (Tugas Mandiri C.3)

| Endpoint & Skenario | Method | `admin` | `staff` | `user` (Owner) | `user` (Non-Owner) | Tanpa Auth | Status Verifikasi |
|---|---|:---:|:---:|:---:|:---:|:---:|:---:|
| `GET /students` (Daftar mahasiswa) | GET | **200** | **200** | **403** | **403** | **401** | ✅ Verified |
| `POST /students` (Tambah mahasiswa) | POST | **201** | **201** | **403** | **403** | **401** | ✅ Verified |
| `GET /students/:id` (Data miliknya) | GET | **200** | **200** | **200** | **403** | **401** | ✅ Verified |
| `GET /students/:id` (Data orang lain) | GET | **200** | **200** | — | **403** | **401** | ✅ Verified |
| `PUT /students/:id` (Edit total miliknya) | PUT | **200** | **403** | **200** | **403** | **401** | ✅ Verified |
| `PUT /students/:id` (Edit data orang lain) | PUT | **200** | **403** | — | **403** | **401** | ✅ Verified |
| `PATCH /students/:id` (Edit parsial miliknya) | PATCH | **200** | **403** | **200** | **403** | **401** | ✅ Verified |
| `PATCH /students/:id` (Edit parsial orang lain) | PATCH | **200** | **403** | — | **403** | **401** | ✅ Verified |
| `DELETE /students/:id` (Hapus mahasiswa) | DELETE | **204** | **403** | **403** | **403** | **401** | ✅ Verified |

---

## 4. BUKTI PENGUJIAN NEGATIF & KASUS KHUSUS

### 1. Percobaan Pemalsuan `owner_id` pada `POST /students`
- **Skenario**: Klien berupaya menyisipkan `"owner_id": 1` di dalam request body `POST /students` untuk mengklaim kepemilikan data atas nama user lain.
- **Perintah Curl**:
  ```bash
  curl -i -X POST http://localhost:3000/api/v1/students \
    -H "Authorization: Bearer $TOKEN_STAFF" \
    -H "Content-Type: application/json" \
    -d '{"nim":"23081010099","name":"Staff Spoofed Owner","grade":85,"owner_id":1}'
  ```
- **Hasil Response (HTTP 201 Created)**:
  ```json
  {
    "success": true,
    "message": "student berhasil dibuat",
    "data": {
      "id": 19,
      "nim": "23081010099",
      "name": "Staff Spoofed Owner",
      "grade": 85,
      "is_active": true,
      "owner_id": 5,
      "created_at": "2026-09-23T21:38:50.236644+07:00"
    }
  }
  ```
- **Analisis**: Kolom `owner_id` pada database tersimpan bernilai `5` (sesuai ID akun staff pembuat token), dan input `"owner_id": 1` dari payload JSON diabaikan sepenuhnya oleh server.

---

### 2. Percobaan Manipulasi Data Milik Orang Lain oleh User Biasa (`IDOR Attack`)
- **Skenario**: User biasa (`budi_owner`, ID 6) memanggil `PUT /students/7` (data milik `andi_user`, ID 7).
- **Perintah Curl**:
  ```bash
  curl -i -X PUT http://localhost:3000/api/v1/students/7 \
    -H "Authorization: Bearer $TOKEN_OWNER" \
    -H "Content-Type: application/json" \
    -d '{"nim":"23081010020","name":"Hacked Data","grade":100,"is_active":true}'
  ```
- **Hasil Response (HTTP 403 Forbidden)**:
  ```json
  {
    "success": false,
    "message": "tidak berhak mengubah data student ini"
  }
  ```
- **Analisis**: Sistem menolak akses karena pemanggil bukan pemilik data (`current.UserID != student.OwnerID`) dan role `user` tidak memiliki hak `student:update:any`.

---

### 3. Percobaan Admin Menurunkan Role Dirinya Sendiri
- **Skenario**: Admin (ID 4) mencoba mengubah role dirinya sendiri menjadi `user` via `PATCH /users/4/role`.
- **Perintah Curl**:
  ```bash
  curl -i -X PATCH http://localhost:3000/api/v1/users/4/role \
    -H "Authorization: Bearer $TOKEN_ADMIN" \
    -H "Content-Type: application/json" \
    -d '{"role":"user"}'
  ```
- **Hasil Response (HTTP 422 Unprocessable Entity)**:
  ```json
  {
    "success": false,
    "message": "validasi gagal",
    "errors": {
      "role": "tidak boleh mengubah role diri sendiri"
    }
  }
  ```
- **Analisis**: Permintaan ditolak dengan status 422 (bukan 403), karena admin memiliki permission `role:assign`, namun tindakannya melanggar *business invariant* untuk mencegah hilangnya akun admin terakhir di sistem.

---

### 4. Percobaan Admin Menghapus Akun Sendiri
- **Skenario**: Admin memanggil `DELETE /users/4` untuk menghapus akun yang sedang digunakan login.
- **Perintah Curl**:
  ```bash
  curl -i -X DELETE http://localhost:3000/api/v1/users/4 \
    -H "Authorization: Bearer $TOKEN_ADMIN"
  ```
- **Hasil Response (HTTP 403 Forbidden)**:
  ```json
  {
    "success": false,
    "message": "tidak boleh menghapus akun sendiri"
  }
  ```

---

### 5. Pembuktian Perilaku Stateless JWT (Token Lama vs Token Baru)
- **Skenario**: Admin mengubah role akun `dummy` dari `user` menjadi `staff` di database. Klien memanggil `GET /students` menggunakan access token yang diterbitkan sebelum kenaikan role.
- **Uji 1 — Menggunakan Token Lama**:
  ```bash
  curl -i http://localhost:3000/api/v1/students \
    -H "Authorization: Bearer $TOKEN_LAMA"
  ```
  **Hasil Response (HTTP 403 Forbidden)**:
  ```json
  {
    "success": false,
    "message": "role user tidak memiliki hak student:list"
  }
  ```
- **Uji 2 — Login Ulang & Menggunakan Token Baru**:
  ```bash
  curl -i http://localhost:3000/api/v1/students \
    -H "Authorization: Bearer $TOKEN_BARU"
  ```
  **Hasil Response (HTTP 200 OK)**:
  ```json
  {
    "success": true,
    "message": "daftar student berhasil diambil",
    "data": [...]
  }
  ```
- **Analisis**: Karena JWT bersifat *stateless*, isi payload role tidak diperiksa ulang ke basis data pada setiap request. Token lama tetap membawa role `user` sampai kadaluarsa. Token baru yang diperoleh setelah login ulang membawa role `staff` dan berhasil mengakses endpoint.

---

## 5. JAWABAN ANALISIS SINGKAT C.4

### 1. Mengapa pemeriksaan kepemilikan tidak dapat dipindahkan ke middleware?
Pemeriksaan kepemilikan (*ownership*) seperti pada fungsi `CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any")` tidak dapat dipindahkan ke lapisan middleware karena middleware hanya memiliki konteks parameter URL (`c.Params("id")`) dan claims JWT pemanggil (`current.UserID`), namun belum mengetahui isi baris data spesifik yang tersimpan di basis data—khususnya kolom `owner_id` dari entitas mahasiswa tersebut. Jika kita memaksakan query ke basis data di dalam middleware untuk mencari `owner_id`, lapisan middleware akan tercemar oleh dependensi basis data dan repository, yang secara langsung melanggar prinsip *Clean Architecture* dan *Separation of Concerns*. Oleh karena itu, pemeriksaan kepemilikan secara arsitektural wajib diletakkan di layer Service (`app/service/student_service.go`), tepat setelah entitas diambil atau sebelum operasi mutasi data dijalankan.

### 2. Sistem Anda kini punya dua tempat pemeriksaan akses: route dan service. Sebutkan satu risiko konkret dari pembagian ini, dan bagaimana Anda mengurangi risiko tersebut.
Risiko konkret dari pemisahan dua lokasi pemeriksaan akses ini adalah **kelalaian pengembang (developer omission / missed enforcement)**, di mana ketika ada penambahan endpoint baru, seorang programmer dapat berasumsi bahwa otorisasi sudah dihandle di middleware sehingga lupa memasang validasi kepemilikan di service, atau sebaliknya mengira otorisasi sudah ada di service sehingga membiarkan route terbuka tanpa penjaga `RequirePermission`. Kami mengurangi risiko tersebut dengan tiga langkah: (1) Menetapkan pola konvensi penamaan permission yang seragam (`domain:action:scope`) serta mendokumentasikan peta hak akses secara eksplisit di komentar `route/route.go`, (2) Mengisolasi logika kepemilikan ke dalam fungsi murni independen (`student_authz_rules.go`) yang diverifikasi secara ketat melalui unit test otomatis, dan (3) Menyusun suite pengujian matriks otomatis (`test_matrix.py`) yang memverifikasi setiap endpoint terhadap seluruh kombinasi role dan relasi kepemilikan.

### 3. Bila suatu hari muncul kebutuhan "dosen wali hanya boleh melihat mahasiswa bimbingannya", apakah RBAC masih memadai? Bila tidak, apa yang perlu ditambahkan?
Model RBAC murni (*Role-Based Access Control*) tidak lagi memadai untuk skenario tersebut karena keputusan hak akses tidak dapat ditentukan semata-mata oleh peran "dosen wali", melainkan bergantung pada **relasi kontekstual dinamis antara atribut identitas subjek (ID Dosen) dan atribut relasi objek (kolom `advisor_id` pada entitas Mahasiswa)**. Untuk mengakomodasi kebutuhan ini, sistem perlu ditingkatkan menjadi kombinasi RBAC dan **ABAC (*Attribute-Based Access Control*) / ReBAC (*Relationship-Based Access Control*)**. Komponen yang perlu ditambahkan meliputi: (1) Penambahan kolom relasional `advisor_id INTEGER REFERENCES users(id)` pada tabel `students`, dan (2) Penambahan aturan evaluasi berbasis atribut pada layer service: `if current.Role == "advisor" && student.AdvisorID == current.UserID { return true }`, sehingga dosen hanya diizinkan membaca baris data mahasiswa yang atribut `advisor_id`-nya identik dengan user ID dosen yang bersangkutan.

---

## 6. JAWABAN EKSPLORASI MANDIRI

1. **Urutan Middleware Terbalik (`RequirePermission` sebelum `RequireAuth`)**:  
   Jika urutan dibalik, `RequirePermission` akan dieksekusi lebih dahulu. Ketika memanggil `helper.CurrentUser(c)`, claims dari token belum diekstrak karena `RequireAuth` belum berjalan. Berdasarkan prinsip *fail closed*, ketiadaan identitas langsung memicu penolakan dengan status **401 Unauthorized ("belum terautentikasi")** bukan 403, karena sistem menyadari bahwa identitas pemanggil belum diketahui sama sekali.

2. **Menghapus Baris `role_permissions` Tanpa Restart Server**:  
   Hasil otorisasi belum berubah karena server membaca pemetaan role-permission dari database **hanya satu kali saat aplikasi booting (startup)** di `main.go` dan menyimpannya di memori (`PermissionSet`). Evaluasi runtime bekerja secara in-memory untuk performa maksimal. Perubahan basis data baru aktif setelah server di-restart.

3. **Menambahkan Role Baru "auditor"**:  
   Dengan pendekatan `RequirePermission`, **0 baris kode Go yang perlu diubah**. Cukup menjalankan 2 baris query SQL (`INSERT INTO roles` dan `INSERT INTO role_permissions`) lalu me-restart aplikasi. Sebaliknya, jika memakai `RequireRole`, pengembang harus mengedit seluruh file route yang memuat daftar role, meng-compile ulang, dan me-redeploy aplikasi.

4. **Salah Ketik Nama Permission pada Route (misal "user:lis")**:  
   Karena prinsip *fail closed*, fungsi `perms.Can(role, "user:lis")` akan mengembalikan `false` untuk semua request, sehingga endpoint langsung membalas **403 Forbidden**. Kesalahan ketik langsung terdeteksi saat pengujian pertama. Jika dirancang *fail open*, salah ketik akan membuat endpoint mengizinkan akses ke seluruh dunia tanpa proteksi.

5. **Memindahkan Pemeriksaan `CanAccessUser` ke Posisi Setelah `FindByID`**:  
   Secara fungsional endpoint tetap menolak pihak yang tidak berhak, namun sistem kehilangan **proteksi terhadap Timing Attack**. Request untuk ID yang ada di DB akan memakan waktu query sebelum ditolak 403, sedangkan ID yang tidak ada akan membalas 404 lebih cepat. Selisih waktu milidetik ini dapat dimanfaatkan penyerang untuk memetakan ID user mana saja yang valid di sistem (*ID enumeration*).

---

## 7. KESIMPULAN

Implementasi sistem otorisasi dan kontrol akses berbasis peran (RBAC) serta kepemilikan data (*Ownership ABAC*) pada proyek ini telah berhasil menyelesaikan seluruh tantangan keamanan akses pada API. Pemisahan tanggung jawab antara Middleware (penjaga gerbang berbasis peran) dan Service (penjaga logika berbasis kepemilikan data) berhasil menjaga kepatuhan terhadap arsitektur *Clean Architecture*. Melalui penerapan prinsip *Fail Closed*, mitigasi *Timing Attack*, dan pengujian matriks komprehensif, sistem kini terlindungi secara kokoh dari kerentanan *Broken Access Control* (OWASP Top 10 A01).
