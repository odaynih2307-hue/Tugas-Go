package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/repository"
)

var (
	ErrForbiddenStudentAccess = errors.New("akses ditolak: mahasiswa hanya dapat mengakses data miliknya sendiri")
)

type StudentService interface {
	List(ctx context.Context, q model.StudentQuery) ([]model.StudentListItem, *model.Meta, error)
	Create(ctx context.Context, req model.CreateStudentRequest) (*model.Student, map[string][]string, int, error)
	GetDetail(ctx context.Context, id int, userRole string, currentStudentID int) (*model.StudentDetailResponse, int, error)
	Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, int, error)
	SoftDelete(ctx context.Context, id int) (int, error)
}

type studentService struct {
	pool           *pgxpool.Pool
	studentRepo    repository.StudentRepository
	userRepo       repository.UserRepository
	enrollmentRepo repository.EnrollmentRepository
}

func NewStudentService(
	pool *pgxpool.Pool,
	studentRepo repository.StudentRepository,
	userRepo repository.UserRepository,
	enrollmentRepo repository.EnrollmentRepository,
) StudentService {
	return &studentService{
		pool:           pool,
		studentRepo:    studentRepo,
		userRepo:       userRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

func (s *studentService) List(ctx context.Context, q model.StudentQuery) ([]model.StudentListItem, *model.Meta, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PerPage <= 0 {
		q.PerPage = 10
	} else if q.PerPage > 50 {
		q.PerPage = 50
	}

	items, total, err := s.studentRepo.FindAll(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	lastPage := 1
	if total > 0 {
		lastPage = int(math.Ceil(float64(total) / float64(q.PerPage)))
	}

	meta := &model.Meta{
		CurrentPage: q.Page,
		PerPage:     q.PerPage,
		Total:       total,
		LastPage:    lastPage,
	}

	return items, meta, nil
}

func (s *studentService) Create(ctx context.Context, req model.CreateStudentRequest) (*model.Student, map[string][]string, int, error) {
	valErrors := make(map[string][]string)

	// Cek apakah NIM sudah terdaftar
	nimExists, err := s.studentRepo.ExistsByNIM(ctx, req.NIM)
	if err != nil {
		return nil, nil, 500, err
	}
	if nimExists {
		valErrors["nim"] = append(valErrors["nim"], "NIM sudah terdaftar")
	}

	// Cek apakah Email sudah terdaftar
	emailExists, err := s.studentRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, 500, err
	}
	if emailExists {
		valErrors["email"] = append(valErrors["email"], "Email sudah terdaftar")
	}

	if len(valErrors) > 0 {
		return nil, valErrors, 422, nil
	}

	// Jalankan dalam satu transaksi basis data
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, 500, fmt.Errorf("memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	// Hash NIM sebagai password awal
	hashedPassword, err := helper.HashPassword(req.NIM)
	if err != nil {
		return nil, nil, 500, fmt.Errorf("hash password mahasiswa: %w", err)
	}

	// 1. Buat record users dengan role mahasiswa
	user, err := s.userRepo.CreateTx(ctx, tx, req.Email, hashedPassword, "mahasiswa")
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			valErrors["email"] = append(valErrors["email"], "Email sudah terdaftar")
			return nil, valErrors, 422, nil
		}
		return nil, nil, 500, err
	}

	// 2. Buat record students
	student, err := s.studentRepo.CreateTx(ctx, tx, req, user.ID)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			valErrors["nim"] = append(valErrors["nim"], "NIM sudah terdaftar")
			return nil, valErrors, 422, nil
		}
		return nil, nil, 500, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, 500, fmt.Errorf("commit transaksi: %w", err)
	}

	return student, nil, 201, nil
}

func (s *studentService) GetDetail(ctx context.Context, id int, userRole string, currentStudentID int) (*model.StudentDetailResponse, int, error) {
	// Jika role mahasiswa, hanya boleh mengakses profil miliknya sendiri
	if userRole == "mahasiswa" && id != currentStudentID {
		return nil, 403, ErrForbiddenStudentAccess
	}

	student, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 404, repository.ErrNotFound
		}
		return nil, 500, err
	}

	// Hitung batas SKS berdasarkan business rule:
	// a. IPK >= 3.00: max 24 SKS
	// b. IPK 2.50 - 2.99: max 21 SKS
	// c. IPK < 2.50: max 18 SKS
	batasSKS := 18
	if student.IPKTerakhir != nil {
		if *student.IPKTerakhir >= 3.00 {
			batasSKS = 24
		} else if *student.IPKTerakhir >= 2.50 {
			batasSKS = 21
		}
	}

	// Ambil daftar mata kuliah yang diambil mahasiswa
	courses, totalSKS, err := s.enrollmentRepo.FindEnrolledCoursesByStudentID(ctx, student.ID)
	if err != nil {
		return nil, 500, err
	}

	detail := &model.StudentDetailResponse{
		ID:          student.ID,
		NIM:         student.NIM,
		Nama:        student.Nama,
		Prodi:       student.Prodi,
		Angkatan:    student.Angkatan,
		IPKTerakhir: student.IPKTerakhir,
		TotalSKS:    totalSKS,
		BatasSKS:    batasSKS,
		MataKuliah:  courses,
	}

	return detail, 200, nil
}

func (s *studentService) Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, int, error) {
	// Pastikan data mahasiswa ada dan belum di-soft delete
	_, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 404, repository.ErrNotFound
		}
		return nil, 500, err
	}

	updated, err := s.studentRepo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 404, repository.ErrNotFound
		}
		return nil, 500, err
	}

	return updated, 200, nil
}

func (s *studentService) SoftDelete(ctx context.Context, id int) (int, error) {
	_, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 404, repository.ErrNotFound
		}
		return 500, err
	}

	if err := s.studentRepo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 404, repository.ErrNotFound
		}
		return 500, err
	}

	return 204, nil
}
