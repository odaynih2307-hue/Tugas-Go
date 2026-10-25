package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type SupervisorService struct {
	supervisors repository.SupervisorRepository
	students    repository.StudentRepository
	tx          *repository.TxManager
}

func NewSupervisorService(
	supervisors repository.SupervisorRepository,
	students repository.StudentRepository,
	tx *repository.TxManager,
) *SupervisorService {
	return &SupervisorService{
		supervisors: supervisors,
		students:    students,
		tx:          tx,
	}
}

// GET /supervisors
func (s *SupervisorService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	list, err := s.supervisors.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.OK(c, "daftar pembimbing berhasil diambil", list)
}

// GET /supervisors/:id
func (s *SupervisorService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	sp, err := s.supervisors.FindByID(ctx, id)
	if err != nil {
		return translateSupervisorError(err)
	}
	return helper.OK(c, "data pembimbing ditemukan", sp)
}

// POST /supervisors/:id/assign
// Menetapkan dosen pembimbing bagi satu mahasiswa.
// Dijaga permission supervision:manage.
// Menggunakan UPDATE bersyarat untuk kuota dan kunci row student (FOR UPDATE)
// untuk mencegah satu mahasiswa ditetapkan ke dua pembimbing secara serentak.
func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id pembimbing harus berupa angka positif")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var assigned model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci baris student secara pesimistik agar request serentak untuk mahasiswa yang sama antre
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Ambil kuota pembimbing secara atomik (UPDATE bersyarat)
		sup, err := repo.IncrementSupervised(ctx, supervisorID)
		if err != nil {
			return err
		}
		assigned = sup

		// 3. Catat penetapan bimbingan. Constraint UNIQUE (student_id) menjamin di level DB
		return repo.InsertSupervision(ctx, req.StudentID, supervisorID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Created(c, "berhasil menetapkan dosen pembimbing", assigned,
		"/api/v1/supervisors/"+strconv.Itoa(supervisorID))
}

// POST /supervisions/switch
// Mengganti pembimbing tugas akhir.
// Wajib atomik (bila pembimbing baru penuh, mahasiswa tetap pada pembimbing lama).
// Wajib bebas deadlock (urutan penguncian seragam berdasar ID pembimbing).
func (s *SupervisorService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var newSupervisor model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci baris student
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Ambil data bimbingan lama
		currentSupervision, err := repo.GetSupervisionByStudent(ctx, req.StudentID)
		if err != nil {
			return err
		}

		fromSupervisorID := currentSupervision.SupervisorID
		toSupervisorID := req.ToSupervisorID

		if fromSupervisorID == toSupervisorID {
			return repository.ErrSameSupervisor
		}

		// 3. Kunci KEDUA pembimbing dengan urutan id menaik (bebas deadlock)
		if err := repo.LockSupervisors(ctx, fromSupervisorID, toSupervisorID); err != nil {
			return err
		}

		// 4. Hapus bimbingan lama
		if err := repo.DeleteSupervision(ctx, req.StudentID); err != nil {
			return err
		}

		// 5. Kurangi jumlah bimbingan pembimbing lama
		if _, err := repo.DecrementSupervised(ctx, fromSupervisorID); err != nil {
			return err
		}

		// 6. Tambah kuota pembimbing baru secara atomik
		sup, err := repo.IncrementSupervised(ctx, toSupervisorID)
		if err != nil {
			return err
		}
		newSupervisor = sup

		// 7. Catat bimbingan baru
		return repo.InsertSupervision(ctx, req.StudentID, toSupervisorID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.OK(c, "berhasil mengganti dosen pembimbing", newSupervisor)
}

// PATCH /supervisors/:id
// Mengubah kuota pembimbing dengan optimistic locking (version column)
func (s *SupervisorService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.supervisors.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.OK(c, "kuota pembimbing berhasil diubah", updated)
}

func translateSupervisorError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing sudah penuh")
	case errors.Is(err, repository.ErrAlreadySupervised):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED", "mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrNoSupervision):
		return helper.NewError(fiber.StatusConflict, "NO_SUPERVISION", "mahasiswa belum memiliki pembimbing")
	case errors.Is(err, repository.ErrSameSupervisor):
		return helper.NewError(fiber.StatusConflict, "SAME_SUPERVISOR", "pembimbing tujuan sama dengan pembimbing saat ini")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT", "data pembimbing sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED", "kuota tidak boleh lebih kecil dari jumlah mahasiswa bimbingan saat ini")
	default:
		return helper.Internal(err)
	}
}
