-- ---------------------------------------------------------------
-- student permissions — hak akses untuk entitas students
-- ---------------------------------------------------------------
INSERT INTO permissions (name, description) VALUES
    ('student:list', 'Melihat daftar mahasiswa'),
    ('student:read:any', 'Melihat data mahasiswa milik siapa pun'),
    ('student:create', 'Menambahkan data mahasiswa baru'),
    ('student:update:any', 'Mengubah data mahasiswa milik siapa pun'),
    ('student:delete', 'Menghapus data mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- ---------------------------------------------------------------
-- role_permissions untuk student
-- ---------------------------------------------------------------
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

-- ---------------------------------------------------------------
-- Menambahkan column owner_id pada tabel students
-- ---------------------------------------------------------------
-- 1. Tambah column jika belum ada
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- 2. Untuk baris data lama yang belum punya owner_id,
-- isi dengan ID user pertama yang ada di tabel users agar tidak melanggar integritas relasi
UPDATE students 
SET owner_id = (SELECT id FROM users ORDER BY id ASC LIMIT 1)
WHERE owner_id IS NULL AND EXISTS (SELECT 1 FROM users);

-- 3. Tambahkan foreign key constraint
ALTER TABLE students DROP CONSTRAINT IF EXISTS fk_students_owner;
ALTER TABLE students
    ADD CONSTRAINT fk_students_owner
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS students_owner_idx ON students (owner_id);
