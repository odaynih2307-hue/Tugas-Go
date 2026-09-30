# LAPORAN PENGERJAAN & PENGUJIAN RESTFUL API SIAKAD MINI
## UJIAN TENGAH SEMESTER (UTS) - PEMROGRAMAN BACKEND GO FIBER

---

### Informasi Tugas
- **Topik:** RESTful API Back End untuk SIAKAD Mini
- **Teknologi:** Go 1.26+, Framework Fiber v2, Driver pgx/v5
- **Basis Data:** PostgreSQL 18.4 (Koneksi Pool `pgxpool`)
- **Keamanan & Otorisasi:** JWT Bearer Token, Bcrypt Hash, Role-Based Access Control (RBAC)
- **Status Pengujian:** 100% PASS (Seluruh 10 Endpoint & Aturan Bisnis Terverifikasi)

---

## 1. Ringkasan Eksekutif & Karakteristik Proyek

SIAKAD Mini adalah layanan akademik backend berbasis RESTful API yang dirancang untuk mengelola data mahasiswa, mata kuliah, dan Kartu Rencana Studi (KRS). Sistem mengimplementasikan **Clean Layered Architecture** yang memisahkan tanggung jawab kode ke dalam lapisan:
1. **Config Layer (`config/`)**: Memuat environment variables dari `.env` secara dinamis.
2. **Database Layer (`database/`)**: Mengelola koneksi pooling `pgxpool` serta migrasi skema dan seeder otomatis.
3. **Model Layer (`app/model/`)**: Mendefinisikan struktur entitas basis data, DTO (Data Transfer Object) request/response, dan meta pagination.
4. **Repository Layer (`app/repository/`)**: Menjalankan query SQL murni dengan parameter binding untuk mencegah SQL Injection, row locking (`SELECT ... FOR UPDATE`), dan soft delete.
5. **Service Layer (`app/service/`)**: Pusat logika bisnis (pengecekan batas SKS berdasarkan IPK, validasi kuota penuh, rate limiting, validasi duplikasi, dan manajemen transaksi atomik).
6. **Handler Layer (`app/handler/`)**: Memvalidasi request payload, memanggil service layer, dan memformat respons JSON seragam.
7. **Middleware Layer (`app/middleware/`)**: Melakukan verifikasi token JWT, ekstraksi klaim peran (Role: Admin / Mahasiswa), proteksi akses tidak sah (401/403), dan pengecekan status soft delete akun.
8. **Helper Layer (`app/helper/`)**: Utilitas pembuatan JWT, hashing password bcrypt, thread-safe in-memory rate limiter, dan format respon seragam.

---

## 2. Struktur Direktori Proyek

Proyek ditempatkan di dalam folder `UTS` pada direktori `latihan-fiber` sesuai instruksi:
```
latihan-fiber/
  UTS/
    app/
      handler/
        auth_handler.go        # Handler Login (429, 401, 422, 200) & Profile /me
        student_handler.go     # Handler CRUD Mahasiswa (List, Create, Get, Update, SoftDelete)
        course_handler.go      # Handler Katalog Mata Kuliah (List with terisi & sisa_kuota)
        enrollment_handler.go  # Handler KRS (Ambil mata kuliah & Batalkan)
      helper/
        jwt.go                 # Generate & Validasi Token JWT
        password.go            # Enkripsi & Komparasi Bcrypt
        ratelimit.go           # Sliding-window Rate Limiter gagal login > 5 kali/menit
        response.go            # Format respon seragam (Success, Error, ValidationError, 500)
        validator.go           # Validasi format email, 12 digit NIM, 4 digit angkatan, dll.
      middleware/
        auth_middleware.go     # Middleware JWT Auth, RequireAdmin, RequireMahasiswa
      model/
        course.go              # Model Course & DTO
        enrollment.go          # Model Enrollment (KRS) & DTO
        response.go            # Struktur APIResponse, ErrorResponse, Meta pagination
        student.go             # Model Student & DTO (Create, Update, Detail)
        user.go                # Model User & Auth DTO
      repository/
        course_repository.go   # Query Mata Kuliah & Row-Locking FOR UPDATE
        enrollment_repository.go # Query KRS & Perhitungan Total SKS
        student_repository.go  # Query Mahasiswa dengan filter, pagination, soft delete
        user_repository.go     # Query User & Pembuatan akun
      service/
        auth_service.go        # Logika login & profil /me
        course_service.go      # Logika katalog mata kuliah
        enrollment_service.go  # Logika KRS: Cek duplikasi, row locking kuota, batas SKS
        student_service.go     # Logika mahasiswa: Transaksi user+student, batas SKS, soft delete
    config/
      config.go              # Pembacaan konfigurasi environment
    database/
      database.go            # PostgreSQL connection pool builder
      seeder.go              # Migrasi & Seeder runner
    migrations/
      001_create_schema.sql  # DDL Tabel users, students, courses, enrollments & indexes
      002_seeder.sql         # Seeder 1 admin, 20 mahasiswa, 10 mata kuliah
    route/
      route.go               # Pendaftaran 10 endpoint RESTful API terpusat
    tests/
      api_test.go            # Automated Test Suite untuk seluruh 10 endpoint
    .env                     # Variabel environment lokal
    .env.example             # Contoh template environment
    go.mod                   # Definisi Go module
    go.sum                   # Checksum dependensi Go
    main.go                  # Entry point aplikasi Fiber
    README.md                # Dokumentasi proyek
    LAPORAN_PENGUJIAN_UTS.md # Laporan pengujian Markdown
    laporan_pengujian_uts.html # Laporan pengujian HTML
    laporan_pengujian_uts.pdf  # Dokumen laporan format PDF resmi
```

---

## 3. Desain Basis Data & Model Relasional

### Tabel Minimal & Relasi:
1. **`users`**:
   - `id`: SERIAL PRIMARY KEY
   - `email`: VARCHAR(255) NOT NULL UNIQUE
   - `password`: VARCHAR(255) NOT NULL (Disimpan dalam bentuk hash Bcrypt)
   - `role`: VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'mahasiswa'))
   - `created_at`, `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
   - *Relasi:* 1–1 dengan tabel `students`

2. **`students`**:
   - `id`: SERIAL PRIMARY KEY
   - `user_id`: INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE
   - `nim`: VARCHAR(12) NOT NULL UNIQUE (12 digit numerik)
   - `nama`: VARCHAR(255) NOT NULL
   - `prodi`: VARCHAR(100) NOT NULL
   - `angkatan`: INT NOT NULL (4 digit, $\le$ tahun berjalan)
   - `ipk_terakhir`: NUMERIC(3,2) CHECK (0.00 $\le$ ipk_terakhir $\le$ 4.00)
   - `deleted_at`: TIMESTAMPTZ DEFAULT NULL (Kolom Soft Delete)
   - `created_at`, `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
   - *Relasi:* 1–N dengan tabel `enrollments`

3. **`courses`**:
   - `id`: SERIAL PRIMARY KEY
   - `kode_mk`: VARCHAR(50) NOT NULL UNIQUE
   - `nama_mk`: VARCHAR(255) NOT NULL
   - `sks`: INT NOT NULL CHECK (sks > 0)
   - `semester`: INT NOT NULL CHECK (semester BETWEEN 1 AND 8)
   - `kuota`: INT NOT NULL CHECK (kuota $\ge$ 0)
   - `created_at`, `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
   - *Relasi:* 1–N dengan tabel `enrollments`

4. **`enrollments`**:
   - `id`: SERIAL PRIMARY KEY
   - `student_id`: INT NOT NULL REFERENCES students(id) ON DELETE CASCADE
   - `course_id`: INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE
   - `tahun_akademik`: VARCHAR(50) NOT NULL (contoh: `2026/2027-Ganjil`)
   - `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
   - *Constraint Unik:* `CONSTRAINT uq_student_course_year UNIQUE (student_id, course_id, tahun_akademik)`

### Data Seeder:
- **1 Admin:** `admin@siakad.ac.id` (password: `admin12345` di-hash dengan bcrypt)
- **20 Mahasiswa:** NIM `187221000001` s/d `187221000020` dengan rentang IPK beragam (IPK $\ge$ 3.00, IPK 2.50–2.99, dan IPK &lt; 2.50) serta password awal = NIM masing-masing yang di-hash dengan bcrypt.
- **10 Mata Kuliah:** Terdistribusi dari semester 1 s/d 7 dengan variasi bobot SKS (2, 3, 4, 6 SKS) dan variasi kuota (termasuk kuota kecil untuk pengujian kuota penuh).

---

## 4. Implementasi Aturan Bisnis (Business Rules)

1. **Batas SKS per Semester Berdasarkan IPK Terakhir:**
   - IPK $\ge$ 3.00 $\rightarrow$ Maksimal **24 SKS**
   - IPK 2.50 – 2.99 $\rightarrow$ Maksimal **21 SKS**
   - IPK &lt; 2.50 (atau `NULL`) $\rightarrow$ Maksimal **18 SKS**
   - *Implementasi:* Dihitung secara real-time pada penambahan KRS dan detail mahasiswa. Jika pengambilan mata kuliah baru menyebabkan total SKS melampaui batas, request ditolak dengan kode status **422 Unprocessable Entity** dan pesan yang secara eksplisit menyebutkan sisa SKS: *"Total SKS melebihi batas (sisa SKS: X, dibutuhkan: Y)"*.

2. **Pencegahan Duplikasi Mata Kuliah di Tahun Akademik Sama:**
   - Mahasiswa tidak dapat mengambil mata kuliah yang sama dua kali pada tahun akademik yang sama.
   - *Implementasi:* Pengecekan sebelum insert dan diproteksi oleh constraint unik database. Pelanggaran mengembalikan kode status **409 Conflict**.

3. **Pencegahan Race Condition Kuota Penuh (Row Locking):**
   - Mata kuliah yang kuotanya penuh (`terisi >= kuota`) tidak dapat diambil.
   - *Implementasi:* Pengambilan mata kuliah dieksekusi dalam satu transaksi database menggunakan **`SELECT ... FOR UPDATE`** pada baris mata kuliah untuk mengunci baris saat kuota dicek. Jika kuota sudah habis, transaksi di-rollback dan mengembalikan kode status **422 Unprocessable Entity**.

4. **Isolasi Hak Akses Mahasiswa (RBAC):**
   - Mahasiswa hanya dapat mengakses profil dan data KRS miliknya sendiri.
   - Mahasiswa hanya dapat membatalkan mata kuliah yang diambil oleh dirinya sendiri.
   - *Implementasi:* Sistem memeriksa ID mahasiswa yang login dari klaim token JWT. Upaya mengakses profil atau menghapus KRS mahasiswa lain ditolak dengan kode status **403 Forbidden**.

5. **Rate Limiting Login Gagal:**
   - Proteksi brute force login: jika terjadi kegagalan login lebih dari 5 kali dalam 1 menit untuk IP/kombinasi akun yang sama, request ditolak dengan kode status **429 Too Many Requests**.

6. **Soft Delete Mahasiswa:**
   - Penghapusan mahasiswa oleh admin hanya mengisi kolom `deleted_at`.
   - Mahasiswa yang dihapus tidak akan muncul pada daftar mahasiswa (Endpoint 3) dan ditolak saat mencoba login dengan kode status **401 Unauthorized**.

7. **Format Respons Seragam:**
   - Respons sukses: `{"success": true, "message": "...", "data": ..., "meta": ...}`
   - Respons error validasi (422): `{"success": false, "message": "Validasi gagal", "errors": {"field": ["alasan"]}}`
   - Respons error umum: `{"success": false, "message": "...", "errors": null}`
   - Respons 204: HTTP Status 204 No Content.
   - Respons 500: Pesan aman tanpa kebocoran stack trace di mode production.

---

## 5. Definisi & Spesifikasi 10 Endpoint

| No | Method | Endpoint | Akses | Fungsi | Status Sukses |
|:--:|:------:|:---------|:-----:|:-------|:-------------:|
| 1 | `POST` | `/api/v1/auth/login` | Publik | Login pengguna, menghasilkan access token JWT | 200 OK |
| 2 | `GET` | `/api/v1/auth/me` | Semua Role | Profil pengguna yang sedang login (beserta detail mahasiswa) | 200 OK |
| 3 | `GET` | `/api/v1/students` | Admin | Daftar mahasiswa (pagination, filter prodi, angkatan, search, sort) | 200 OK |
| 4 | `POST` | `/api/v1/students` | Admin | Menambah mahasiswa & akun user (password = NIM di-hash) dalam 1 transaksi | 201 Created |
| 5 | `GET` | `/api/v1/students/{id}` | Admin, Mahasiswa (sendiri) | Detail mahasiswa, daftar mata kuliah, total SKS & batas SKS | 200 OK |
| 6 | `PUT` | `/api/v1/students/{id}` | Admin | Memperbarui data mahasiswa (NIM tidak boleh diubah) | 200 OK |
| 7 | `DELETE` | `/api/v1/students/{id}` | Admin | Soft delete mahasiswa (isi `deleted_at`) | 204 No Content |
| 8 | `GET` | `/api/v1/courses` | Semua Role | Daftar mata kuliah dengan kalkulasi kolom `terisi` & `sisa_kuota` | 200 OK |
| 9 | `POST` | `/api/v1/enrollments` | Mahasiswa | Mengambil mata kuliah ke KRS (transaksi, row lock, batas SKS) | 201 Created |
| 10 | `DELETE` | `/api/v1/enrollments/{id}` | Mahasiswa (sendiri) | Membatalkan mata kuliah dari KRS | 204 No Content |

---

## 6. Laporan Rinci Hasil Pengujian (Testing Report)

Pengujian dilakukan menggunakan suite pengujian otomatis berbasis Go (`go test -v ./tests`). Berikut adalah penjelasan lengkap per skenario pengujian:

### 🔹 Pengujian Endpoint 1: POST `/api/v1/auth/login`
- **1.1 Admin Login Sukses (Status: 200 OK):**
  - *Input:* `{"email": "admin@siakad.ac.id", "password": "admin12345"}`
  - *Ekspektasi:* HTTP 200 OK, payload memuat `access_token`, `token_type: "Bearer"`, `expires_in: 3600`, dan data user dengan `role: "admin"`.
  - *Hasil:* **PASS**. Token berhasil diekstrak dan diverifikasi.
- **1.2 Mahasiswa Login Sukses (Status: 200 OK):**
  - *Input:* Email mahasiswa (`rina.putri@siakad.ac.id`) dan password = NIM (`187221000001`).
  - *Ekspektasi:* HTTP 200 OK, menghasilkan token JWT mahasiswa.
  - *Hasil:* **PASS**.
- **1.3 Login Gagal - Kredensial Salah (Status: 401 Unauthorized):**
  - *Input:* Email valid namun password salah.
  - *Ekspektasi:* HTTP 401 dengan pesan error kredensial tidak valid.
  - *Hasil:* **PASS**.
- **1.4 Login Gagal - Validasi Input (Status: 422 Unprocessable Entity):**
  - *Input:* Format email tidak valid (`"bukan-email"`) dan password kurang dari 8 karakter (`"123"`).
  - *Ekspektasi:* HTTP 422 dengan objek `errors` berisi rincian validasi per-field.
  - *Hasil:* **PASS**.
- **1.5 Rate Limiting Login Gagal > 5 Kali (Status: 429 Too Many Requests):**
  - *Input:* Menjalankan percobaan login dengan kredensial salah sebanyak 6 kali berturut-turut dalam 1 menit.
  - *Ekspektasi:* Percobaan ke-6 langsung ditolak dengan HTTP 429 Too Many Requests.
  - *Hasil:* **PASS**.

### 🔹 Pengujian Endpoint 2: GET `/api/v1/auth/me`
- **2.1 Admin Mengakses /me (Status: 200 OK):**
  - *Header:* `Authorization: Bearer <token_admin>`
  - *Hasil:* **PASS**. Mengembalikan profil user admin tanpa password.
- **2.2 Mahasiswa Mengakses /me (Status: 200 OK):**
  - *Header:* `Authorization: Bearer <token_mahasiswa>`
  - *Hasil:* **PASS**. Mengembalikan profil user dan menyertakan objek `student` berisi NIM, Nama, Prodi, Angkatan, dan IPK.
- **2.3 Tanpa Token Otorisasi (Status: 401 Unauthorized):**
  - *Header:* Tanpa header Authorization.
  - *Hasil:* **PASS**. HTTP 401 Unauthorized.

### 🔹 Pengujian Endpoint 3: GET `/api/v1/students`
- **3.1 Admin Melihat Daftar Mahasiswa Berhalaman (Status: 200 OK):**
  - *Query Params:* `page=1&per_page=10`
  - *Hasil:* **PASS**. Mengembalikan 10 data mahasiswa dan objek `meta` (`current_page: 1, per_page: 10, total: 20, last_page: 2`).
- **3.2 Filter Search & Sorting (Status: 200 OK):**
  - *Query Params:* `search=Rina&sort=-ipk_terakhir`
  - *Hasil:* **PASS**. Berhasil memfilter nama mahasiswa dan mengurutkan berdasarkan IPK tertinggi.
- **3.3 Otorisasi Non-Admin (Status: 403 Forbidden):**
  - *Header:* Menggunakan token mahasiswa.
  - *Hasil:* **PASS**. Ditolak dengan HTTP 403 Forbidden.

### 🔹 Pengujian Endpoint 4: POST `/api/v1/students`
- **4.1 Admin Menambah Mahasiswa Baru (Status: 201 Created):**
  - *Input:* NIM 12 digit, nama, email, prodi, angkatan, IPK.
  - *Proses:* Transaksi database membuat record `users` dan `students`.
  - *Hasil:* **PASS**. Mengembalikan data mahasiswa baru, dan akun baru berhasil diuji login menggunakan NIM sebagai password awal.
- **4.2 Validasi Duplikasi NIM & Email (Status: 422 Unprocessable Entity):**
  - *Input:* Mengirimkan NIM dan email yang sama dengan data yang baru dibuat.
  - *Hasil:* **PASS**. Mengembalikan HTTP 422 dengan pesan `"NIM sudah terdaftar"` dan `"Email sudah terdaftar"`.
- **4.3 Mahasiswa Menambah Mahasiswa (Status: 403 Forbidden):**
  - *Header:* Menggunakan token mahasiswa.
  - *Hasil:* **PASS**. HTTP 403 Forbidden.

### 🔹 Pengujian Endpoint 5: GET `/api/v1/students/{id}`
- **5.1 Mahasiswa Mengakses Data Milik Sendiri (Status: 200 OK):**
  - *Kondisi:* Rina Putri (IPK 3.45) mengakses profil miliknya sendiri.
  - *Hasil:* **PASS**. Mengembalikan profil mahasiswa, daftar mata kuliah diambil, `total_sks`, dan `batas_sks = 24`.
- **5.2 Mahasiswa Mengakses Data Mahasiswa Lain (Status: 403 Forbidden):**
  - *Kondisi:* Rina Putri mencoba mengakses ID mahasiswa milik Budi Santoso.
  - *Hasil:* **PASS**. Ditolak dengan HTTP 403 Forbidden ("Akses ditolak: mahasiswa hanya dapat mengakses profil miliknya sendiri").
- **5.3 Admin Mengakses Data Mahasiswa (Status: 200 OK):**
  - *Hasil:* **PASS**. Admin bebas melihat detail data mahasiswa manapun.

### 🔹 Pengujian Endpoint 6: PUT `/api/v1/students/{id}`
- **6.1 Admin Memperbarui Data Mahasiswa (Status: 200 OK):**
  - *Input:* Nama baru, prodi baru, angkatan baru, IPK baru.
  - *Hasil:* **PASS**. Data berhasil diperbarui di basis data tanpa mengubah NIM.
- **6.2 Mahasiswa Mencoba Memperbarui Data (Status: 403 Forbidden):**
  - *Hasil:* **PASS**. Ditolak dengan HTTP 403 Forbidden.

### 🔹 Pengujian Endpoint 7: DELETE `/api/v1/students/{id}`
- **7.1 Admin Melakukan Soft Delete (Status: 204 No Content):**
  - *Hasil:* **PASS**. Mengembalikan HTTP 204 No Content dan kolom `deleted_at` terisi timestamp.
- **7.2 Verifikasi Tidak Muncul di List:**
  - *Hasil:* **PASS**. Mahasiswa yang telah di-soft delete tidak lagi muncul di GET `/api/v1/students`.
- **7.3 Verifikasi Mahasiswa Terhapus Ditolak Login (Status: 401 Unauthorized):**
  - *Hasil:* **PASS**. Upaya login akun mahasiswa terhapus ditolak dengan HTTP 401 Unauthorized ("akun mahasiswa tidak aktif atau telah dinonaktifkan").

### 🔹 Pengujian Endpoint 8: GET `/api/v1/courses`
- **8.1 Mengambil Daftar Mata Kuliah (Status: 200 OK):**
  - *Hasil:* **PASS**. Mengembalikan minimal 10 mata kuliah dengan kolom `terisi` dan `sisa_kuota` terhitung secara akurat dari tabel enrollments (`sisa_kuota = kuota - terisi`).
- **8.2 Filter available=true:**
  - *Hasil:* **PASS**. Hanya menampilkan mata kuliah yang kuotanya masih tersedia.

### 🔹 Pengujian Endpoint 9: POST `/api/v1/enrollments`
- **9.1 Mahasiswa Mengambil Mata Kuliah ke KRS (Status: 201 Created):**
  - *Input:* `course_id: 1`, `tahun_akademik: "2026/2027-Ganjil"`.
  - *Hasil:* **PASS**. Record tersimpan di tabel enrollments.
- **9.2 Pencegahan Duplikasi Pengambilan Mata Kuliah (Status: 409 Conflict):**
  - *Input:* Mengambil kembali `course_id: 1` pada tahun akademik yang sama.
  - *Hasil:* **PASS**. Ditolak dengan HTTP 409 Conflict ("Mata kuliah sudah pernah diambil pada tahun akademik ini").
- **9.3 Admin Mengambil Mata Kuliah (Status: 403 Forbidden):**
  - *Hasil:* **PASS**. Hanya role mahasiswa yang diizinkan mengambil mata kuliah ke KRS.
- **9.4 Validasi Batas SKS Terlampaui (Status: 422 Unprocessable Entity):**
  - *Kondisi:* Mahasiswa Tiara Andini (IPK 1.95 $\rightarrow$ batas 18 SKS) yang telah mengambil 17 SKS mencoba mengambil mata kuliah berbobot 3 SKS (total 20 SKS > 18 SKS).
  - *Hasil:* **PASS**. Ditolak dengan HTTP 422 dan pesan eksplisit: *"Total SKS melebihi batas (sisa SKS: 1, dibutuhkan: 3)"*.
- **9.5 Validasi Kuota Penuh dengan Row Locking (Status: 422 Unprocessable Entity):**
  - *Kondisi:* Mata kuliah IF402 memiliki kuota = 2. Setelah diisi penuh oleh dua mahasiswa, mahasiswa ketiga mencoba mengambil mata kuliah tersebut.
  - *Hasil:* **PASS**. Ditolak dengan HTTP 422 Unprocessable Entity ("Kuota mata kuliah sudah penuh").

### 🔹 Pengujian Endpoint 10: DELETE `/api/v1/enrollments/{id}`
- **10.1 Mahasiswa Membatalkan KRS Milik Mahasiswa Lain (Status: 403 Forbidden):**
  - *Kondisi:* Mahasiswa B mencoba menghapus enrollment milik Mahasiswa A.
  - *Hasil:* **PASS**. Ditolak dengan HTTP 403 Forbidden ("Akses ditolak: Anda tidak dapat membatalkan mata kuliah milik mahasiswa lain").
- **10.2 Mahasiswa Membatalkan KRS Milik Sendiri (Status: 204 No Content):**
  - *Hasil:* **PASS**. Record enrollment terhapus, kuota mata kuliah otomatis kembali bertambah, dan respon menghasilkan HTTP 204 No Content.
- **10.3 Pembatalan ID Tidak Ditemukan (Status: 404 Not Found):**
  - *Hasil:* **PASS**. Mengembalikan HTTP 404 Not Found.

---

## 7. Matriks Status Code HTTP yang Dipersyaratkan

| Status Code | Definisi | Implementasi Endpoint & Skenario | Hasil Uji |
|:-----------:|:---------|:---------------------------------|:---------:|
| **200** | OK | Login sukses, profil me, daftar mahasiswa, detail mahasiswa, katalog mata kuliah, update mahasiswa | **PASS** |
| **201** | Created | Pembuatan mahasiswa & akun baru (POST students), pengambilan KRS (POST enrollments) | **PASS** |
| **204** | No Content | Soft delete mahasiswa (DELETE students), pembatalan mata kuliah KRS (DELETE enrollments) | **PASS** |
| **401** | Unauthorized | Kredensial salah, token tidak ada / kedaluwarsa, mahasiswa soft-deleted mencoba login | **PASS** |
| **403** | Forbidden | Mahasiswa akses fitur admin, akses detail mahasiswa lain, batalkan KRS mahasiswa lain | **PASS** |
| **404** | Not Found | ID mahasiswa / mata kuliah / enrollment tidak ditemukan atau sudah dihapus | **PASS** |
| **409** | Conflict | Mahasiswa mengambil mata kuliah yang sama dua kali pada tahun akademik yang sama | **PASS** |
| **422** | Unprocessable Entity | Validasi gagal, NIM/Email duplikat, kuota mata kuliah penuh, total SKS melebihi batas | **PASS** |
| **429** | Too Many Requests | Rate limiting: Gagal login lebih dari 5 kali dalam 1 menit | **PASS** |
| **500** | Internal Server Error | Penanganan error server internal tanpa kebocoran stack trace pada mode production | **PASS** |

---

## 8. Panduan Eksekusi & Pengujian Mandiri

1. **Pastikan Basis Data Siap:**
   ```bash
   # Buat database jika belum ada
   psql -U postgres -c "CREATE DATABASE siakad_mini;"
   ```
2. **Jalankan Migrasi & Seeder Awal:**
   ```bash
   cd latihan-fiber/UTS
   go run main.go -seed
   ```
3. **Jalankan Server API:**
   ```bash
   go run main.go
   ```
4. **Jalankan Pengujian Otomatis:**
   ```bash
   go test -v ./tests
   ```
   *Output yang diharapkan:* Semua 32 sub-test case lulus (`PASS`) dengan waktu eksekusi < 2 detik.
