# API Students

REST API sederhana untuk mengelola data mahasiswa menggunakan Go dan Fiber.

## Base URL

http://localhost:3000

## Endpoint

| Method | Endpoint | Keterangan |
|---|---|---|
| GET | `/api/v1/students` | Mengambil daftar student |
| GET | `/api/v1/students/:id` | Mengambil student berdasarkan ID |
| POST | `/api/v1/students` | Menambahkan student |
| PUT | `/api/v1/students/:id` | Memperbarui seluruh data student |
| PATCH | `/api/v1/students/:id` | Memperbarui sebagian data student |
| DELETE | `/api/v1/students/:id` | Menghapus student |

## Query Parameter

Endpoint GET `/api/v1/students` mendukung:

- `page` — nomor halaman
- `limit` — jumlah data per halaman
- `search` — mencari berdasarkan nama
- `sort` — field untuk sorting
- `order` — urutan `asc` atau `desc`
- `is_active` — filter status aktif

// Contoh Query:
// GET /api/v1/students?page=1&limit=2
// GET /api/v1/students?search=and
// GET /api/v1/students?sort=grade&order=desc
// GET /api/v1/students?is_active=true
// GET /api/v1/students?is_active=true&sort=grade&order=desc&page=1&limit=1