# API Students — Database & Repository Pattern

REST API untuk manajemen data mahasiswa (Students) yang dibangun menggunakan **Go**, framework **Fiber v2**, driver **pgx/v5 (connection pool)**, serta menerapkan **Repository Pattern** dan database **PostgreSQL**.

---

## 📋 Daftar Variabel Environment

Aplikasi ini membaca konfigurasi melalui file `.env`. Silakan salin template dari `.env.example`:

```bash
cp .env.example .env
```

Isi variabel-variabel berikut di file `.env`:

| Variabel | Deskripsi | Contoh Nilai |
|---|---|---|
| `APP_PORT` | Port server aplikasi | `3000` |
| `DB_HOST` | Host database PostgreSQL | `localhost` |
| `DB_PORT` | Port database PostgreSQL | `5432` |
| `DB_USER` | Username PostgreSQL | `postgres` |
| `DB_PASSWORD` | Password user PostgreSQL | `123456` |
| `DB_NAME` | Nama database yang digunakan | `api_students_db` |
| `DB_SSLMODE` | Mode enkripsi SSL | `disable` |
| `DB_MAX_CONNS` | Maksimum koneksi connection pool | `10` |

---

## 🗄️ Skema Tabel Basis Data

File migrasi berada di: `migrations/001_create_students.sql`

```sql
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    grade NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indeks unik untuk NIM (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key
    ON students (LOWER(nim));

-- Indeks kolom nama untuk mempercepat query ILIKE
CREATE INDEX IF NOT EXISTS students_name_lower_idx
    ON students (LOWER(name));
```

---

## 🚀 Cara Menyiapkan dan Menjalankan Proyek dari Nol

### 1. Buat Database di PostgreSQL
Buka terminal dan jalankan:
```bash
psql -U postgres -c "CREATE DATABASE api_students_db;"
```

### 2. Jalankan Migrasi Skema Tabel
Jalankan file SQL migrasi untuk membuat tabel dan indeks:
```bash
psql -U postgres -d api_students_db -f migrations/001_create_students.sql
```

### 3. Setup Dependencies
Unduh package Go yang dibutuhkan:
```bash
go mod tidy
```

### 4. Jalankan Aplikasi
```bash
go run .
```
Server akan berjalan di `http://localhost:3000`.

---

## 📡 Daftar Endpoint API

### Base URL
`http://localhost:3000/api/v1`

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/health` | Memeriksa status kesehatan server dan koneksi database |
| GET | `/students` | Mengambil daftar mahasiswa dengan filter, sorting, dan paginasi |
| GET | `/students/:id` | Mengambil data mahasiswa berdasarkan ID |
| POST | `/students` | Menambahkan mahasiswa baru |
| PUT | `/students/:id` | Memperbarui seluruh data mahasiswa |
| PATCH | `/students/:id` | Memperbarui sebagian field data mahasiswa |
| DELETE | `/students/:id` | Menghapus data mahasiswa berdasarkan ID |

### Query Parameters untuk `GET /api/v1/students`
- `page` (int, default: 1): Nomor halaman
- `limit` (int, default: 10, max: 100): Jumlah data per halaman
- `search` (string): Mencari data berdasarkan substring nama (ILIKE)
- `sort` (string, default: "id"): Kolom pengurutan (`id`, `nim`, `name`, `grade`, `created_at`)
- `order` (string, default: "asc"): Arah pengurutan (`asc` / `desc`)
- `is_active` (bool): Filter status aktif (`true` / `false`)