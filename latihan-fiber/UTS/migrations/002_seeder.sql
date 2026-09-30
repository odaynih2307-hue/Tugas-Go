-- Seeder for SIAKAD Mini
-- Requirement: minimal 1 admin, 20 mahasiswa, 10 mata kuliah
-- All passwords hashed with bcrypt

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Seed Admin User
INSERT INTO users (email, password, role)
VALUES ('admin@siakad.ac.id', crypt('admin12345', gen_salt('bf', 10)), 'admin')
ON CONFLICT (email) DO NOTHING;

-- 2. Seed 20 Mahasiswa (Users + Students)
-- Mahasiswa 1
INSERT INTO users (email, password, role)
VALUES ('rina.putri@siakad.ac.id', crypt('187221000001', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'rina.putri@siakad.ac.id'), '187221000001', 'Rina Putri', 'Sistem Informasi', 2022, 3.45)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 2
INSERT INTO users (email, password, role)
VALUES ('budi.santoso@siakad.ac.id', crypt('187221000002', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'budi.santoso@siakad.ac.id'), '187221000002', 'Budi Santoso', 'Teknik Informatika', 2021, 3.85)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 3
INSERT INTO users (email, password, role)
VALUES ('siti.nurhaliza@siakad.ac.id', crypt('187221000003', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'siti.nurhaliza@siakad.ac.id'), '187221000003', 'Siti Nurhaliza', 'Sains Data', 2023, 2.85)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 4
INSERT INTO users (email, password, role)
VALUES ('ahmad.fauzi@siakad.ac.id', crypt('187221000004', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'ahmad.fauzi@siakad.ac.id'), '187221000004', 'Ahmad Fauzi', 'Teknik Komputer', 2022, 2.30)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 5
INSERT INTO users (email, password, role)
VALUES ('dewi.lestari@siakad.ac.id', crypt('187221000005', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'dewi.lestari@siakad.ac.id'), '187221000005', 'Dewi Lestari', 'Sistem Informasi', 2021, 3.70)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 6
INSERT INTO users (email, password, role)
VALUES ('rizky.pratama@siakad.ac.id', crypt('187221000006', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'rizky.pratama@siakad.ac.id'), '187221000006', 'Rizky Pratama', 'Teknik Informatika', 2023, 2.75)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 7
INSERT INTO users (email, password, role)
VALUES ('anisa.rahma@siakad.ac.id', crypt('187221000007', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'anisa.rahma@siakad.ac.id'), '187221000007', 'Anisa Rahmawati', 'Sains Data', 2022, 3.90)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 8
INSERT INTO users (email, password, role)
VALUES ('fajar.hidayat@siakad.ac.id', crypt('187221000008', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'fajar.hidayat@siakad.ac.id'), '187221000008', 'Fajar Hidayat', 'Teknik Komputer', 2024, 2.40)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 9
INSERT INTO users (email, password, role)
VALUES ('maya.anggraini@siakad.ac.id', crypt('187221000009', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'maya.anggraini@siakad.ac.id'), '187221000009', 'Maya Anggraini', 'Teknik Informatika', 2021, 3.25)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 10
INSERT INTO users (email, password, role)
VALUES ('dimas.prasetyo@siakad.ac.id', crypt('187221000010', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'dimas.prasetyo@siakad.ac.id'), '187221000010', 'Dimas Prasetyo', 'Sistem Informasi', 2022, 2.65)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 11
INSERT INTO users (email, password, role)
VALUES ('nadia.safitri@siakad.ac.id', crypt('187221000011', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'nadia.safitri@siakad.ac.id'), '187221000011', 'Nadia Safitri', 'Sains Data', 2023, 3.10)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 12
INSERT INTO users (email, password, role)
VALUES ('ilham.ramadhan@siakad.ac.id', crypt('187221000012', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'ilham.ramadhan@siakad.ac.id'), '187221000012', 'Ilham Ramadhan', 'Teknik Informatika', 2022, 2.95)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 13
INSERT INTO users (email, password, role)
VALUES ('putri.wulandari@siakad.ac.id', crypt('187221000013', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'putri.wulandari@siakad.ac.id'), '187221000013', 'Putri Wulandari', 'Teknik Komputer', 2024, 2.15)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 14
INSERT INTO users (email, password, role)
VALUES ('hendra.wijaya@siakad.ac.id', crypt('187221000014', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'hendra.wijaya@siakad.ac.id'), '187221000014', 'Hendra Wijaya', 'Sistem Informasi', 2021, 3.60)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 15
INSERT INTO users (email, password, role)
VALUES ('gita.gutawa@siakad.ac.id', crypt('187221000015', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'gita.gutawa@siakad.ac.id'), '187221000015', 'Gita Gutawa', 'Sains Data', 2023, 2.88)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 16
INSERT INTO users (email, password, role)
VALUES ('bayu.saputra@siakad.ac.id', crypt('187221000016', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'bayu.saputra@siakad.ac.id'), '187221000016', 'Bayu Saputra', 'Teknik Informatika', 2022, 3.50)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 17
INSERT INTO users (email, password, role)
VALUES ('tiara.andini@siakad.ac.id', crypt('187221000017', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'tiara.andini@siakad.ac.id'), '187221000017', 'Tiara Andini', 'Teknik Komputer', 2024, 1.95)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 18
INSERT INTO users (email, password, role)
VALUES ('kevin.sanjaya@siakad.ac.id', crypt('187221000018', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'kevin.sanjaya@siakad.ac.id'), '187221000018', 'Kevin Sanjaya', 'Sistem Informasi', 2023, 3.35)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 19
INSERT INTO users (email, password, role)
VALUES ('clarissa.aurelia@siakad.ac.id', crypt('187221000019', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'clarissa.aurelia@siakad.ac.id'), '187221000019', 'Clarissa Aurelia', 'Teknik Informatika', 2022, 2.55)
ON CONFLICT (nim) DO NOTHING;

-- Mahasiswa 20
INSERT INTO users (email, password, role)
VALUES ('zidan.alfarizi@siakad.ac.id', crypt('187221000020', gen_salt('bf', 10)), 'mahasiswa')
ON CONFLICT (email) DO NOTHING;
INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
VALUES ((SELECT id FROM users WHERE email = 'zidan.alfarizi@siakad.ac.id'), '187221000020', 'Zidan Alfarizi', 'Sains Data', 2023, 3.05)
ON CONFLICT (nim) DO NOTHING;

-- 3. Seed 10 Mata Kuliah
INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota) VALUES
('IF101', 'Algoritma dan Pemrograman', 3, 1, 30),
('IF102', 'Struktur Data dan Algoritma', 4, 2, 25),
('IF201', 'Basis Data Relasional', 3, 3, 30),
('IF202', 'Pemrograman Web Lanjut', 3, 3, 20),
('IF301', 'Rekayasa Perangkat Lunak', 3, 4, 25),
('IF302', 'Jaringan Komputer', 3, 4, 25),
('IF401', 'Kecerdasan Buatan', 3, 5, 20),
('IF402', 'Pemrograman Backend Go Fiber', 4, 5, 2),
('IF501', 'Keamanan Sistem Informasi', 3, 6, 15),
('IF502', 'Skripsi dan Tugas Akhir', 6, 7, 50)
ON CONFLICT (kode_mk) DO NOTHING;

-- 4. Initial Enrollments (Optional seed for testing course calculation)
-- Mahasiswa 2 (Budi Santoso) takes IF101 & IF201
INSERT INTO enrollments (student_id, course_id, tahun_akademik)
VALUES
((SELECT id FROM students WHERE nim = '187221000002'), (SELECT id FROM courses WHERE kode_mk = 'IF101'), '2026/2027-Ganjil'),
((SELECT id FROM students WHERE nim = '187221000002'), (SELECT id FROM courses WHERE kode_mk = 'IF201'), '2026/2027-Ganjil')
ON CONFLICT DO NOTHING;
