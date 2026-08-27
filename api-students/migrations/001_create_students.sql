CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    grade NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Keunikan NIM tanpa membedakan huruf besar dan kecil.
-- Mencegah pendaftaran ganda secara konkruen di level basis data.
CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key
    ON students (LOWER(nim));

-- Indeks pada LOWER(name) untuk mempercepat pencarian (ILIKE).
CREATE INDEX IF NOT EXISTS students_name_lower_idx
    ON students (LOWER(name));
