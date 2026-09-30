package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"uts-siakad/app/model"
)

type StudentRepository interface {
	FindAll(ctx context.Context, q model.StudentQuery) ([]model.StudentListItem, int, error)
	FindByID(ctx context.Context, id int) (*model.Student, error)
	FindByUserID(ctx context.Context, userID int) (*model.Student, error)
	CreateTx(ctx context.Context, tx pgx.Tx, req model.CreateStudentRequest, userID int) (*model.Student, error)
	Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, error)
	SoftDelete(ctx context.Context, id int) error
	ExistsByNIM(ctx context.Context, nim string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.StudentQuery) ([]model.StudentListItem, int, error) {
	whereClauses := []string{"deleted_at IS NULL"}
	args := []any{}
	argIdx := 1

	if q.Prodi != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("prodi ILIKE $%d", argIdx))
		args = append(args, "%"+q.Prodi+"%")
		argIdx++
	}

	if q.Angkatan > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("angkatan = $%d", argIdx))
		args = append(args, q.Angkatan)
		argIdx++
	}

	if q.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(nim ILIKE $%d OR nama ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+q.Search+"%")
		argIdx++
	}

	whereSQL := " WHERE " + strings.Join(whereClauses, " AND ")

	// Hitung total data
	var total int
	countQuery := "SELECT COUNT(*) FROM students" + whereSQL
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung total mahasiswa: %w", err)
	}

	// Tentukan klausa ORDER BY
	orderBy := "id ASC"
	switch q.Sort {
	case "nama":
		orderBy = "nama ASC"
	case "-nama":
		orderBy = "nama DESC"
	case "ipk_terakhir":
		orderBy = "ipk_terakhir ASC NULLS LAST"
	case "-ipk_terakhir":
		orderBy = "ipk_terakhir DESC NULLS LAST"
	}

	limit := q.PerPage
	offset := q.Offset()
	selectQuery := fmt.Sprintf(
		`SELECT id, nim, nama, prodi, angkatan, ipk_terakhir
		 FROM students
		 %s
		 ORDER BY %s
		 LIMIT $%d OFFSET $%d`,
		whereSQL, orderBy, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar mahasiswa: %w", err)
	}
	defer rows.Close()

	items := []model.StudentListItem{}
	for rows.Next() {
		var item model.StudentListItem
		if err := rows.Scan(&item.ID, &item.NIM, &item.Nama, &item.Prodi, &item.Angkatan, &item.IPKTerakhir); err != nil {
			return nil, 0, fmt.Errorf("membaca row mahasiswa: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi row mahasiswa: %w", err)
	}

	return items, total, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (*model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at, updated_at
		 FROM students
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari mahasiswa berdasarkan id: %w", err)
	}

	return &s, nil
}

func (r *studentPostgresRepository) FindByUserID(ctx context.Context, userID int) (*model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at, updated_at
		 FROM students
		 WHERE user_id = $1 AND deleted_at IS NULL`, userID,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari mahasiswa berdasarkan user_id: %w", err)
	}

	return &s, nil
}

func (r *studentPostgresRepository) CreateTx(ctx context.Context, tx pgx.Tx, req model.CreateStudentRequest, userID int) (*model.Student, error) {
	var s model.Student
	err := tx.QueryRow(ctx,
		`INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at`,
		userID, req.NIM, req.Nama, req.Prodi, req.Angkatan, req.IPKTerakhir,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("menyimpan mahasiswa baru: %w", err)
	}

	return &s, nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`UPDATE students
		 SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4, updated_at = NOW()
		 WHERE id = $5 AND deleted_at IS NULL
		 RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at`,
		req.Nama, req.Prodi, req.Angkatan, req.IPKTerakhir, id,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPKTerakhir, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("memperbarui mahasiswa: %w", err)
	}

	return &s, nil
}

func (r *studentPostgresRepository) SoftDelete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE students
		 SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("soft delete mahasiswa: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *studentPostgresRepository) ExistsByNIM(ctx context.Context, nim string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM students WHERE nim = $1 AND deleted_at IS NULL)`, nim,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cek keberadaan NIM: %w", err)
	}
	return exists, nil
}

func (r *studentPostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cek keberadaan Email: %w", err)
	}
	return exists, nil
}
