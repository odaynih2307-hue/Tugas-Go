-- Migration 009: Pembimbing Tugas Akhir (Supervisors & Supervisions)
CREATE TABLE IF NOT EXISTS supervisors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    quota INTEGER NOT NULL CHECK (quota > 0),
    supervised_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supervisors_supervised_within_quota
    CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    id BIGSERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL UNIQUE REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_idx ON supervisions (supervisor_id);

INSERT INTO permissions (name, description) VALUES
    ('supervision:manage', 'Menetapkan dan mengelola dosen pembimbing tugas akhir')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;

-- Seed supervisors for testing
INSERT INTO supervisors (id, name, quota, supervised_count, version) VALUES
    (1, 'Dr. Ir. Budi Santoso, M.Kom.', 3, 0, 1),
    (2, 'Prof. Dr. Siti Rahmawati, S.T., M.T.', 5, 0, 1),
    (3, 'Dr. Eng. Ahmad Dahlan, M.Eng.', 3, 0, 1)
ON CONFLICT (id) DO NOTHING;
SELECT setval('supervisors_id_seq', (SELECT COALESCE(MAX(id), 1) FROM supervisors));

-- Seed 50 students for load testing if not already present
INSERT INTO students (nim, name, grade, is_active)
SELECT '112233' || LPAD(g::TEXT, 4, '0'),
       'Mahasiswa Uji ' || LPAD(g::TEXT, 3, '0'),
       3.50,
       TRUE
FROM generate_series(1, 50) AS g
ON CONFLICT DO NOTHING;
