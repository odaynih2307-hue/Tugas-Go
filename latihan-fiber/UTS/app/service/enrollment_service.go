package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"uts-siakad/app/model"
	"uts-siakad/app/repository"
)

var (
	ErrAlreadyEnrolled         = errors.New("mata kuliah sudah pernah diambil pada tahun akademik ini")
	ErrCourseQuotaFull         = errors.New("kuota mata kuliah sudah penuh")
	ErrForbiddenEnrollmentDrop = errors.New("akses ditolak: tidak dapat membatalkan mata kuliah milik mahasiswa lain")
)

type EnrollmentService interface {
	Enroll(ctx context.Context, studentID int, req model.CreateEnrollmentRequest) (*model.Enrollment, int, string, error)
	Delete(ctx context.Context, enrollmentID int, currentStudentID int) (int, string, error)
}

type enrollmentService struct {
	pool           *pgxpool.Pool
	enrollmentRepo repository.EnrollmentRepository
	courseRepo     repository.CourseRepository
	studentRepo    repository.StudentRepository
}

func NewEnrollmentService(
	pool *pgxpool.Pool,
	enrollmentRepo repository.EnrollmentRepository,
	courseRepo repository.CourseRepository,
	studentRepo repository.StudentRepository,
) EnrollmentService {
	return &enrollmentService{
		pool:           pool,
		enrollmentRepo: enrollmentRepo,
		courseRepo:     courseRepo,
		studentRepo:    studentRepo,
	}
}

func (s *enrollmentService) Enroll(ctx context.Context, studentID int, req model.CreateEnrollmentRequest) (*model.Enrollment, int, string, error) {
	// Pastikan data mahasiswa aktif
	student, err := s.studentRepo.FindByID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 404, "Data mahasiswa tidak ditemukan atau telah dinonaktifkan", err
		}
		return nil, 500, "Gagal mengambil data mahasiswa", err
	}

	// Hitung batas SKS mahasiswa berdasarkan IPK terakhir
	batasSKS := 18
	if student.IPKTerakhir != nil {
		if *student.IPKTerakhir >= 3.00 {
			batasSKS = 24
		} else if *student.IPKTerakhir >= 2.50 {
			batasSKS = 21
		}
	}

	// Jalankan proses validasi dan penyisipan dalam satu transaksi dengan row locking
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, 500, "Gagal memulai transaksi", err
	}
	defer tx.Rollback(ctx)

	// 1. Cek duplikasi pengambilan mata kuliah pada tahun akademik yang sama
	already, err := s.enrollmentRepo.IsAlreadyEnrolledTx(ctx, tx, studentID, req.CourseID, req.TahunAkademik)
	if err != nil {
		return nil, 500, "Gagal memeriksa riwayat pengambilan mata kuliah", err
	}
	if already {
		return nil, 409, "Mata kuliah sudah pernah diambil pada tahun akademik ini", ErrAlreadyEnrolled
	}

	// 2. Kunci baris mata kuliah (SELECT ... FOR UPDATE) untuk mencegah race condition kuota
	course, err := s.courseRepo.FindByIDForUpdateTx(ctx, tx, req.CourseID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 404, "Mata kuliah tidak ditemukan", err
		}
		return nil, 500, "Gagal mengunci baris mata kuliah", err
	}

	// 3. Periksa kuota mata kuliah
	enrolledCount, err := s.enrollmentRepo.GetEnrolledCountByCourseIDTx(ctx, tx, req.CourseID)
	if err != nil {
		return nil, 500, "Gagal menghitung kuota terisi", err
	}
	if enrolledCount >= course.Kuota {
		return nil, 422, "Kuota mata kuliah sudah penuh", ErrCourseQuotaFull
	}

	// 4. Periksa batas SKS mahasiswa untuk semester/tahun akademik tersebut
	currentTotalSKS, err := s.enrollmentRepo.GetTotalSKSByStudentAndYearTx(ctx, tx, studentID, req.TahunAkademik)
	if err != nil {
		return nil, 500, "Gagal menghitung total SKS terambil", err
	}

	if currentTotalSKS+course.SKS > batasSKS {
		sisaSKS := batasSKS - currentTotalSKS
		if sisaSKS < 0 {
			sisaSKS = 0
		}
		msg := fmt.Sprintf("Total SKS melebihi batas (sisa SKS: %d, dibutuhkan: %d)", sisaSKS, course.SKS)
		return nil, 422, msg, errors.New(msg)
	}

	// 5. Simpan record enrollment
	enrollment, err := s.enrollmentRepo.CreateTx(ctx, tx, studentID, req.CourseID, req.TahunAkademik)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, 409, "Mata kuliah sudah pernah diambil pada tahun akademik ini", ErrAlreadyEnrolled
		}
		return nil, 500, "Gagal menyimpan pengambilan mata kuliah", err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 500, "Gagal menyelesaikan transaksi", err
	}

	return enrollment, 201, "Mata kuliah berhasil ditambahkan ke KRS", nil
}

func (s *enrollmentService) Delete(ctx context.Context, enrollmentID int, currentStudentID int) (int, string, error) {
	enrollment, err := s.enrollmentRepo.FindByID(ctx, enrollmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 404, "Data pengambilan mata kuliah tidak ditemukan", err
		}
		return 500, "Gagal mencari data pengambilan mata kuliah", err
	}

	// Mahasiswa hanya dapat membatalkan KRS miliknya sendiri
	if enrollment.StudentID != currentStudentID {
		return 403, "Akses ditolak: Anda tidak dapat membatalkan mata kuliah milik mahasiswa lain", ErrForbiddenEnrollmentDrop
	}

	if err := s.enrollmentRepo.Delete(ctx, enrollmentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 404, "Data pengambilan mata kuliah tidak ditemukan", err
		}
		return 500, "Gagal menghapus data pengambilan mata kuliah", err
	}

	return 204, "Mata kuliah berhasil dibatalkan dari KRS", nil
}
