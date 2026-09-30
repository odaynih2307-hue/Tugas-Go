package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"uts-siakad/app/model"
)

type EnrollmentRepository interface {
	FindEnrolledCoursesByStudentID(ctx context.Context, studentID int) ([]model.EnrolledCourseItem, int, error)
	IsAlreadyEnrolledTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (bool, error)
	GetEnrolledCountByCourseIDTx(ctx context.Context, tx pgx.Tx, courseID int) (int, error)
	GetTotalSKSByStudentAndYearTx(ctx context.Context, tx pgx.Tx, studentID int, tahunAkademik string) (int, error)
	CreateTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error)
	FindByID(ctx context.Context, id int) (*model.Enrollment, error)
	Delete(ctx context.Context, id int) error
}

type enrollmentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) EnrollmentRepository {
	return &enrollmentPostgresRepository{pool: pool}
}

func (r *enrollmentPostgresRepository) FindEnrolledCoursesByStudentID(ctx context.Context, studentID int) ([]model.EnrolledCourseItem, int, error) {
	query := `
		SELECT e.id, c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, e.tahun_akademik, e.created_at
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE e.student_id = $1
		ORDER BY e.id ASC
	`
	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil mata kuliah mahasiswa: %w", err)
	}
	defer rows.Close()

	items := []model.EnrolledCourseItem{}
	totalSKS := 0
	for rows.Next() {
		var item model.EnrolledCourseItem
		if err := rows.Scan(&item.EnrollmentID, &item.CourseID, &item.KodeMK, &item.NamaMK, &item.SKS, &item.Semester, &item.TahunAkademik, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca row mata kuliah diambil: %w", err)
		}
		items = append(items, item)
		totalSKS += item.SKS
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi row mata kuliah diambil: %w", err)
	}

	return items, totalSKS, nil
}

func (r *enrollmentPostgresRepository) IsAlreadyEnrolledTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM enrollments
			WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3
		)`, studentID, courseID, tahunAkademik,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cek duplikasi pengambilan mata kuliah: %w", err)
	}
	return exists, nil
}

func (r *enrollmentPostgresRepository) GetEnrolledCountByCourseIDTx(ctx context.Context, tx pgx.Tx, courseID int) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1`, courseID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("menghitung jumlah kuota terisi: %w", err)
	}
	return count, nil
}

func (r *enrollmentPostgresRepository) GetTotalSKSByStudentAndYearTx(ctx context.Context, tx pgx.Tx, studentID int, tahunAkademik string) (int, error) {
	var total int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.sks), 0)
		 FROM enrollments e
		 JOIN courses c ON e.course_id = c.id
		 WHERE e.student_id = $1 AND e.tahun_akademik = $2`,
		studentID, tahunAkademik,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("menghitung total SKS terambil: %w", err)
	}
	return total, nil
}

func (r *enrollmentPostgresRepository) CreateTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error) {
	var e model.Enrollment
	err := tx.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, course_id, tahun_akademik, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, student_id, course_id, tahun_akademik, created_at`,
		studentID, courseID, tahunAkademik,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("menyimpan enrollment: %w", err)
	}

	return &e, nil
}

func (r *enrollmentPostgresRepository) FindByID(ctx context.Context, id int) (*model.Enrollment, error) {
	var e model.Enrollment
	err := r.pool.QueryRow(ctx,
		`SELECT id, student_id, course_id, tahun_akademik, created_at
		 FROM enrollments
		 WHERE id = $1`, id,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari enrollment: %w", err)
	}

	return &e, nil
}

func (r *enrollmentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM enrollments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
