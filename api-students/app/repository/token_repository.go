package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token tidak ditemukan")

type RefreshToken struct {
	ID        int
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{
		pool: pool,
	}
}

func (r *TokenRepository) Create(
	ctx context.Context,
	userID int,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)

	if err != nil {
		return fmt.Errorf("gagal menyimpan refresh token: %w", err)
	}

	return nil
}

func (r *TokenRepository) FindActiveByHash(
	ctx context.Context,
	tokenHash string,
) (*RefreshToken, error) {
	var token RefreshToken

	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}

		return nil, fmt.Errorf("gagal mencari refresh token: %w", err)
	}

	if token.RevokedAt != nil || !token.ExpiresAt.After(time.Now()) {
		return nil, ErrRefreshTokenNotFound
	}

	return &token, nil
}

func (r *TokenRepository) Revoke(
	ctx context.Context,
	id int,
) error {
	now := time.Now()

	result, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE id = $2
		  AND revoked_at IS NULL
	`, now, id)

	if err != nil {
		return fmt.Errorf("gagal mencabut refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
