package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"uts-siakad/app/model"
)

type CourseRepository interface {
	FindAll(ctx context.Context, q model.CourseQuery) ([]model.CourseResponse, error)
	FindByID(ctx context.Context, id int) (*model.Course, error)
	FindByIDForUpdateTx(ctx context.Context, tx pgx.Tx, id int) (*model.Course, error)
}

type coursePostgresRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &coursePostgresRepository{pool: pool}
}

func (r *coursePostgresRepository) FindAll(ctx context.Context, q model.CourseQuery) ([]model.CourseResponse, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if q.Semester > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.semester = $%d", argIdx))
		args = append(args, q.Semester)
		argIdx++
	}

	if q.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+q.Search+"%")
		argIdx++
	}

	whereSQL := " WHERE " + strings.Join(whereClauses, " AND ")

	havingSQL := ""
	if q.Available {
		havingSQL = " HAVING (c.kuota - COUNT(e.id)) > 0"
	}

	querySQL := fmt.Sprintf(
		`SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
		        COUNT(e.id) AS terisi,
		        (c.kuota - COUNT(e.id)) AS sisa_kuota
		 FROM courses c
		 LEFT JOIN enrollments e ON e.course_id = c.id
		 %s
		 GROUP BY c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota
		 %s
		 ORDER BY c.id ASC`,
		whereSQL, havingSQL,
	)

	rows, err := r.pool.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar mata kuliah: %w", err)
	}
	defer rows.Close()

	courses := []model.CourseResponse{}
	for rows.Next() {
		var c model.CourseResponse
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi, &c.SisaKuota); err != nil {
			return nil, fmt.Errorf("membaca row mata kuliah: %w", err)
		}
		courses = append(courses, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi row mata kuliah: %w", err)
	}

	return courses, nil
}

func (r *coursePostgresRepository) FindByID(ctx context.Context, id int) (*model.Course, error) {
	var c model.Course
	err := r.pool.QueryRow(ctx,
		`SELECT id, kode_mk, nama_mk, sks, semester, kuota, created_at, updated_at
		 FROM courses
		 WHERE id = $1`, id,
	).Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari mata kuliah berdasarkan id: %w", err)
	}

	return &c, nil
}

// FindByIDForUpdateTx melakukan row locking menggunakan SELECT ... FOR UPDATE di dalam transaksi.
func (r *coursePostgresRepository) FindByIDForUpdateTx(ctx context.Context, tx pgx.Tx, id int) (*model.Course, error) {
	var c model.Course
	err := tx.QueryRow(ctx,
		`SELECT id, kode_mk, nama_mk, sks, semester, kuota, created_at, updated_at
		 FROM courses
		 WHERE id = $1
		 FOR UPDATE`, id,
	).Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mengambil mata kuliah dengan row lock: %w", err)
	}

	return &c, nil
}
