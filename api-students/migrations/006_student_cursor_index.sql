-- -----------------------------------------------------------------------------
-- Migration 006: Index Keyset Cursor Pagination untuk Tabel Students (Tugas D.3)
-- -----------------------------------------------------------------------------
-- Pasangan kolom pengurut: (created_at DESC, id DESC).
-- Alasan pemilihan:
-- created_at tidak dijamin unik (beberapa mahasiswa dapat di-insert dalam satu
-- mikrodetik transaksi yang sama). Penambahan primary key 'id' berfungsi sebagai
-- tie-breaker deterministik yang menjamin seluruh entitas memiliki urutan unik mutlak.
-- Arah DESC memastikan halaman awal memuat mahasiswa yang paling baru didaftarkan.

CREATE INDEX IF NOT EXISTS students_created_at_id_desc_idx
 ON students (created_at DESC, id DESC);
