package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

const (
	LocalsUserID   = "user_id"
	LocalsUsername = "username"
	LocalsRole     = "role"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"`
	User         *model.User `json:"user"`
}

type AuthService struct {
	userRepository  *repository.UserRepository
	tokenRepository *repository.TokenRepository
	jwtManager      *helper.JWTManager
	perms           *helper.PermissionSet
	refreshTTL      time.Duration
}

func NewAuthService(
	userRepository *repository.UserRepository,
	tokenRepository *repository.TokenRepository,
	jwtManager *helper.JWTManager,
	perms *helper.PermissionSet,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepository:  userRepository,
		tokenRepository: tokenRepository,
		jwtManager:      jwtManager,
		perms:           perms,
		refreshTTL:      refreshTTL,
	}
}

// Register membuat akun baru menggunakan validasi deklaratif (Langkah 6).
// Role selalu ditentukan server agar user tidak bisa mendaftarkan dirinya sebagai admin.
func (s *AuthService) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format JSON tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Role selalu ditentukan server.
	role := "user"

	passwordHash, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	user, err := s.userRepository.Create(
		c.Context(),
		req.Username,
		passwordHash,
		role,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return helper.Conflict("username sudah digunakan")
		}

		return helper.Internal(err)
	}

	return helper.Created(
		c,
		"user berhasil didaftarkan",
		s.safeUser(user),
		"/api/v1/auth/me",
	)
}

// Login menghasilkan access token dan refresh token.
func (s *AuthService) Login(c *fiber.Ctx) error {
	var req LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request tidak valid")
	}

	req.Username = strings.TrimSpace(req.Username)

	user, err := s.userRepository.FindByUsername(
		c.Context(),
		req.Username,
	)

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Tetap melakukan bcrypt compare agar perbedaan waktu antara
			// username ada dan username tidak ada tidak terlalu mudah ditebak.
			_ = helper.CheckPassword(
				req.Password,
				"$2a$12$C6UzMDM.H6dfI/f/IKcEe.9j2j9Q1wXxY0K8N6kM4Z0Z1Q4Y7fJQ2",
			)

			return helper.Unauthorized("username atau password salah")
		}

		return helper.Internal(err)
	}

	if !helper.CheckPassword(req.Password, user.Password) {
		return helper.Unauthorized("username atau password salah")
	}

	accessToken, expiresIn, err := s.jwtManager.GenerateAccessToken(
		user.ID,
		user.Username,
		user.Role,
	)
	if err != nil {
		return helper.Internal(err)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return helper.Internal(err)
	}

	refreshTokenHash := hashRefreshToken(refreshToken)
	refreshExpiresAt := time.Now().Add(s.refreshTTL)

	if err := s.tokenRepository.Create(
		c.Context(),
		user.ID,
		refreshTokenHash,
		refreshExpiresAt,
	); err != nil {
		return helper.Internal(err)
	}

	return helper.OK(
		c,
		"login berhasil",
		TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    expiresIn,
			User:         s.safeUser(user),
		},
	)
}

// Refresh melakukan rotation refresh token.
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request tidak valid")
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)

	if req.RefreshToken == "" {
		return helper.Unauthorized("refresh token tidak valid")
	}

	tokenHash := hashRefreshToken(req.RefreshToken)

	oldToken, err := s.tokenRepository.FindActiveByHash(
		c.Context(),
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return helper.Unauthorized("refresh token tidak valid atau sudah tidak aktif")
		}

		return helper.Internal(err)
	}

	user, err := s.userRepository.FindByID(
		c.Context(),
		oldToken.UserID,
	)
	if err != nil {
		return helper.Unauthorized("refresh token tidak valid")
	}

	accessToken, expiresIn, err := s.jwtManager.GenerateAccessToken(
		user.ID,
		user.Username,
		user.Role,
	)
	if err != nil {
		return helper.Internal(err)
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return helper.Internal(err)
	}

	newRefreshHash := hashRefreshToken(newRefreshToken)
	newRefreshExpiresAt := time.Now().Add(s.refreshTTL)

	if err := s.tokenRepository.Revoke(
		c.Context(),
		oldToken.ID,
	); err != nil {
		return helper.Internal(err)
	}

	if err := s.tokenRepository.Create(
		c.Context(),
		user.ID,
		newRefreshHash,
		newRefreshExpiresAt,
	); err != nil {
		return helper.Internal(err)
	}

	return helper.OK(
		c,
		"refresh token berhasil dirotasi",
		TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    expiresIn,
			User:         s.safeUser(user),
		},
	)
}

// Logout mencabut refresh token yang sedang digunakan.
func (s *AuthService) Logout(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request tidak valid")
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)

	if req.RefreshToken == "" {
		return helper.Unauthorized("refresh token tidak valid")
	}

	tokenHash := hashRefreshToken(req.RefreshToken)

	token, err := s.tokenRepository.FindActiveByHash(
		c.Context(),
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return helper.Unauthorized("refresh token tidak valid atau sudah tidak aktif")
		}

		return helper.Internal(err)
	}

	if err := s.tokenRepository.Revoke(
		c.Context(),
		token.ID,
	); err != nil {
		return helper.Internal(err)
	}

	return helper.OK(
		c,
		"logout berhasil",
		nil,
	)
}

// Me mengambil identitas user dari middleware authentication.
func (s *AuthService) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals(LocalsUserID).(int)
	if !ok {
		return helper.Unauthorized("akses tidak terautentikasi")
	}

	user, err := s.userRepository.FindByID(
		c.Context(),
		userID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return helper.Unauthorized("user tidak ditemukan")
		}

		return helper.Internal(err)
	}

	permissions := []string{}
	if s.perms != nil {
		permissions = s.perms.PermissionsOf(user.Role)
	}

	return helper.OK(
		c,
		"profil user berhasil diambil",
		fiber.Map{
			"user":        s.safeUser(user),
			"permissions": permissions,
		},
	)
}

// safeUser memastikan password hash tidak pernah dikirim ke response.
func (s *AuthService) safeUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}

	return &model.User{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}

func generateRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
