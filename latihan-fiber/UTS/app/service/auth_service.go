package service

import (
	"context"
	"errors"

	"uts-siakad/app/helper"
	"uts-siakad/app/model"
	"uts-siakad/app/repository"
)

var (
	ErrInvalidCredentials = errors.New("kredensial login tidak valid")
	ErrAccountInactive    = errors.New("akun mahasiswa tidak aktif atau telah dinonaktifkan")
	ErrRateLimited        = errors.New("gagal login lebih dari 5 kali per menit, silakan tunggu sesaat")
)

type AuthService interface {
	Login(ctx context.Context, req model.LoginRequest, clientIP string) (*model.LoginResponseData, int, error)
	Me(ctx context.Context, userID int) (*model.AuthMeResponse, error)
}

type authService struct {
	userRepo    repository.UserRepository
	studentRepo repository.StudentRepository
	limiter     *helper.LoginRateLimiter
}

func NewAuthService(userRepo repository.UserRepository, studentRepo repository.StudentRepository, limiter *helper.LoginRateLimiter) AuthService {
	return &authService{
		userRepo:    userRepo,
		studentRepo: studentRepo,
		limiter:     limiter,
	}
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest, clientIP string) (*model.LoginResponseData, int, error) {
	rateKey := clientIP + ":" + req.Email

	// 1. Cek apakah IP/akun sedang terkena rate limiting (> 5 kegagalan dalam 1 menit)
	if s.limiter.IsRateLimited(rateKey) {
		return nil, 429, ErrRateLimited
	}

	// 2. Cari user berdasarkan email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			if s.limiter.RecordFailure(rateKey) {
				return nil, 429, ErrRateLimited
			}
			return nil, 401, ErrInvalidCredentials
		}
		return nil, 500, err
	}

	// 3. Verifikasi password hash bcrypt
	if !helper.CheckPasswordHash(req.Password, user.Password) {
		if s.limiter.RecordFailure(rateKey) {
			return nil, 429, ErrRateLimited
		}
		return nil, 401, ErrInvalidCredentials
	}

	// 4. Jika user mahasiswa, pastikan tidak soft-deleted
	var studentID int
	if user.Role == "mahasiswa" {
		student, err := s.studentRepo.FindByUserID(ctx, user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				if s.limiter.RecordFailure(rateKey) {
					return nil, 429, ErrRateLimited
				}
				return nil, 401, ErrAccountInactive
			}
			return nil, 500, err
		}
		studentID = student.ID
	}

	// Login berhasil, bersihkan penghitung rate limit
	s.limiter.Reset(rateKey)

	// 5. Generate JWT token
	token, expiresIn, err := helper.GenerateToken(user.ID, user.Email, user.Role, studentID)
	if err != nil {
		return nil, 500, err
	}

	return &model.LoginResponseData{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User: model.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, 200, nil
}

func (s *authService) Me(ctx context.Context, userID int) (*model.AuthMeResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &model.AuthMeResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	if user.Role == "mahasiswa" {
		student, err := s.studentRepo.FindByUserID(ctx, user.ID)
		if err == nil && student != nil {
			resp.Student = &model.StudentProfile{
				ID:          student.ID,
				NIM:         student.NIM,
				Nama:        student.Nama,
				Prodi:       student.Prodi,
				Angkatan:    student.Angkatan,
				IPKTerakhir: student.IPKTerakhir,
			}
		}
	}

	return resp, nil
}
