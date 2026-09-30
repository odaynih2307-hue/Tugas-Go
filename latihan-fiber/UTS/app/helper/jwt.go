package helper

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"uts-siakad/config"
)

// JWTClaims memuat klaim khusus untuk payload JWT SIAKAD Mini.
type JWTClaims struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	StudentID int    `json:"student_id,omitempty"`
	jwt.RegisteredClaims
}

var (
	ErrTokenInvalid = errors.New("token tidak valid atau telah kedaluwarsa")
	ErrTokenMissing = errors.New("token otorisasi tidak ditemukan")
)

// GenerateToken membuat access token JWT baru berdasarkan data user.
func GenerateToken(userID int, email, role string, studentID int) (string, int, error) {
	ttl := time.Duration(config.AppConfig.JWTExpiresIn) * time.Second
	expiresAt := time.Now().Add(ttl)

	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		StudentID: studentID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.AppConfig.JWTIssuer,
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return "", 0, fmt.Errorf("menandatangani token: %w", err)
	}

	return tokenString, int(ttl.Seconds()), nil
}

// ValidateToken memvalidasi string JWT dan mengembalikan pointer ke JWTClaims.
func ValidateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode signing tidak diharapkan: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
