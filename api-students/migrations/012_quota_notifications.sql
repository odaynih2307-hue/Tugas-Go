-- Migration 012: Penggantian Idempotency Key untuk Mendukung Notifikasi Multi-User (Modul 11 Tugas Mandiri D.2)
--
-- Ketika event seperti perubahan kuota kelas terjadi, satu baris outbox (outbox_id)
-- akan menghasilkan banyak notifikasi (satu untuk setiap mahasiswa yang terdaftar di kelas tersebut).
--
-- Constraint UNIQUE (outbox_id) sebelumnya membatasi hanya satu notifikasi per event outbox.
-- Oleh karena itu, kita menggantinya dengan UNIQUE (outbox_id, user_id) sehingga:
-- 1. Satu event outbox dapat mengirimkan notifikasi ke banyak mahasiswa berbeda.
-- 2. Namun, jika terjadi kegagalan/retry di level worker, setiap mahasiswa tetap dijamin
--    menerima TEPAT SATU notifikasi untuk event tersebut (idempoten per user).

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);
