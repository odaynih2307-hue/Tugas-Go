package helper

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret    string
	issuer    string
	accessTTL time.Duration
}

type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTManager(
	secret string,
	issuer string,
	accessTTL time.Duration,
) *JWTManager {
	return &JWTManager{
		secret:    secret,
		issuer:    issuer,
		accessTTL: accessTTL,
	}
}

func (m *JWTManager) GenerateAccessToken(
	userID int,
	username string,
	role string,
) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)

	claims := AccessClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", 0, err
	}

	return signedToken, int64(m.accessTTL.Seconds()), nil
}

func (m *JWTManager) ValidateAccessToken(
	tokenString string,
) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AccessClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Hanya izinkan algoritma HS256.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}

			return []byte(m.secret), nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
