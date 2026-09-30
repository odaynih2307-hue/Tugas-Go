# SIAKAD Mini - RESTful API Backend (UTS)

SIAKAD Mini adalah layanan backend akademik sederhana berbasis RESTful API yang mengelola data mahasiswa, mata kuliah, dan Kartu Rencana Studi (KRS) sesuai ketentuan ujian tengah semester (UTS).

Dibangun menggunakan bahasa **Go**, framework **Fiber v2**, driver database **pgx/v5**, dan basis data **PostgreSQL**.

---

## 🌟 Fitur Utama & Ketentuan Teknis

1. **Teknologi**:
   - Bahasa: **Go 1.26+**
   - Web Framework: **Fiber v2** (`github.com/gofiber/fiber/v2`)
   - Driver Basis Data: **pgx/v5** (`github.com/jackc/pgx/v5/pgxpool`)
   - Basis Data: **PostgreSQL 16+**
   - Autentikasi: **JWT (JSON Web Token)** (`github.com/golang-jwt/jwt/v5`)
   - Keamanan Password: **Bcrypt Hash** (`golang.org/x/crypto/bcrypt`)

2. **Dua Peran Pengguna (RBAC)**:
   - **Admin**: Mengelola data mahasiswa (CRUD lengkap dengan soft delete) dan melihat mata kuliah.
   - **Mahasiswa**: Melihat profil dan KRS sendiri, mengambil mata kuliah (menambah KRS), serta membatalkan mata kuliah dari KRS miliknya.

3. **Aturan Bisnis (Business Rules)**:
   - **Batas SKS per semester**:
     - IPK $\ge$ 3.00: Maksimal 24 SKS
     - IPK 2.50 – 2.99: Maksimal 21 SKS
     - IPK < 2.50 (atau belum ada): Maksimal 18 SKS
   - **Pencegahan Duplikasi**: Mahasiswa tidak dapat mengambil mata kuliah yang sama dua kali pada tahun akademik yang sama (Error 409 Conflict).
   - **Pencegahan Race Condition Kuota**: Menggunakan *Row-Level Locking* (`SELECT ... FOR UPDATE`) dalam satu database transaction. Mata kuliah yang kuotanya penuh ditolak (Error 422).
   - **Otorisasi Data Pribadi**: Mahasiswa hanya dapat melihat profil dan membatalkan KRS miliknya sendiri. Akses ke data mahasiswa lain ditolak (Error 403 Forbidden).
   - **Rate Limiting Login**: Gagal login lebih dari 5 kali per menit diblokir sementara (Error 429 Too Many Requests).
   - **Soft Delete**: Mahasiswa yang di-soft delete tidak muncul di daftar mahasiswa dan ditolak saat mencoba login (Error 401 Unauthorized).

---

## 📋 Daftar 10 Endpoint

| No | Method | Endpoint | Akses | Fungsi | Status Sukses |
|:--:|:------:|:---------|:-----:|:-------|:-------------:|
| 1 | `POST` | `/api/v1/auth/login` | Publik | Autentikasi login, mengembalikan JWT access token | 200 OK |
| 2 | `GET` | `/api/v1/auth/me` | Semua Role | Profil pengguna login (termasuk ringkasan mahasiswa) | 200 OK |
| 3 | `GET` | `/api/v1/students` | Admin | Daftar mahasiswa (pagination, filter prodi/angkatan, search, sorting) | 200 OK |
| 4 | `POST` | `/api/v1/students` | Admin | Menambah mahasiswa baru & akun user dalam satu transaksi | 201 Created |
| 5 | `GET` | `/api/v1/students/{id}` | Admin, Mahasiswa (data sendiri) | Detail mahasiswa beserta total SKS & daftar mata kuliah | 200 OK |
| 6 | `PUT` | `/api/v1/students/{id}` | Admin | Memperbarui data mahasiswa (NIM tidak boleh diubah) | 200 OK |
| 7 | `DELETE` | `/api/v1/students/{id}` | Admin | Soft delete mahasiswa (isi deleted_at) | 204 No Content |
| 8 | `GET` | `/api/v1/courses` | Semua Role | Daftar mata kuliah beserta kuota `terisi` & `sisa_kuota` | 200 OK |
| 9 | `POST` | `/api/v1/enrollments` | Mahasiswa | Mengambil mata kuliah (menambah KRS) | 201 Created |
| 10 | `DELETE` | `/api/v1/enrollments/{id}` | Mahasiswa (milik sendiri) | Membatalkan mata kuliah dari KRS | 204 No Content |

---

## 🗄️ Model Data Minimal

1. **`users`**:
   - `id` (SERIAL PRIMARY KEY)
   - `email` (VARCHAR, UNIQUE)
   - `password` (VARCHAR, Bcrypt hash)
   - `role` (VARCHAR: `admin` / `mahasiswa`)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
2. **`students`**:
   - `id` (SERIAL PRIMARY KEY)
   - `user_id` (INT, 1-1 REFERENCES users(id))
   - `nim` (VARCHAR(12), UNIQUE)
   - `nama` (VARCHAR)
   - `prodi` (VARCHAR)
   - `angkatan` (INT)
   - `ipk_terakhir` (NUMERIC(3,2), range 0.00 - 4.00)
   - `deleted_at` (TIMESTAMPTZ, NULLable untuk soft delete)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
3. **`courses`**:
   - `id` (SERIAL PRIMARY KEY)
   - `kode_mk` (VARCHAR, UNIQUE)
   - `nama_mk` (VARCHAR)
   - `sks` (INT > 0)
   - `semester` (INT 1-8)
   - `kuota` (INT >= 0)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
4. **`enrollments`**:
   - `id` (SERIAL PRIMARY KEY)
   - `student_id` (INT REFERENCES students(id))
   - `course_id` (INT REFERENCES courses(id))
   - `tahun_akademik` (VARCHAR, misal `2026/2027-Ganjil`)
   - `created_at` (TIMESTAMPTZ)
   - `UNIQUE (student_id, course_id, tahun_akademik)`

---

## 🚀 Panduan Menjalankan

### 1. Konfigurasi Environment (`.env`)
Salin file `.env.example` menjadi `.env` lalu sesuaikan kredensial PostgreSQL Anda:
```env
APP_PORT=3000
APP_ENV=development
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=123456
DB_NAME=siakad_mini
DB_SSLMODE=disable
DB_MAX_CONNS=10

JWT_SECRET=supersecretjwtkeywithmorethan32bytes123456!
JWT_ISSUER=siakad-mini
JWT_EXPIRES_IN=3600
```

### 2. Migrasi & Seeder Database
Pastikan database PostgreSQL `siakad_mini` telah dibuat di PostgreSQL:
```bash
# Menjalankan migrasi skema dan seeder (1 admin, 20 mahasiswa, 10 mata kuliah)
go run main.go -seed
```

### 3. Menjalankan Server API
```bash
go run main.go
```
Server akan aktif di `http://localhost:3000`.

### 4. Menjalankan Suite Pengujian Otomatis (Unit/Integration Tests)
```bash
go test -v ./tests
```
Semua 10 endpoint dan aturan bisnis (termasuk status code 200, 201, 204, 401, 403, 404, 409, 422, 429, dan 500) akan diuji secara otomatis dan tervalidasi 100% PASS.

---

## 🔑 Akun Uji Coba Awal (Seeder)

- **Admin**:
  - Email: `admin@siakad.ac.id`
  - Password: `admin12345`
- **Mahasiswa (20 Mahasiswa)**:
  - Rina Putri: `rina.putri@siakad.ac.id` / Password: `187221000001` (NIM)
  - Budi Santoso: `budi.santoso@siakad.ac.id` / Password: `187221000002` (NIM)
  - Siti Nurhaliza: `siti.nurhaliza@siakad.ac.id` / Password: `187221000003` (NIM)
  - ... hingga Mahasiswa ke-20 (`187221000020`)
