package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
)

var (
	ErrUserNotFound = errors.New("user tidak ditemukan")
	ErrUserExists   = errors.New("username sudah digunakan")
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	username string,
	passwordHash string,
	role string,
) (*model.User, error) {
	var user model.User

	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO users (username, password, role)
		VALUES ($1, $2, $3)
		RETURNING id, username, password, role, created_at
		`,
		username,
		passwordHash,
		role,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}

		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (*model.User, error) {
	var user model.User

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT id, username, password, role, created_at
		FROM users
		WHERE username = $1
		`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("gagal mencari user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id int,
) (*model.User, error) {
	var user model.User

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT id, username, password, role, created_at
		FROM users
		WHERE id = $1
		`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("gagal mencari user: %w", err)
	}

	return &user, nil
}
