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

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data duplikat atau sudah terdaftar")
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	CreateTx(ctx context.Context, tx pgx.Tx, email, hashedPassword, role string) (*model.User, error)
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func (r *userPostgresRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, role, created_at, updated_at
		 FROM users
		 WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari user berdasarkan email: %w", err)
	}

	return &u, nil
}

func (r *userPostgresRepository) FindByID(ctx context.Context, id int) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, role, created_at, updated_at
		 FROM users
		 WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("mencari user berdasarkan id: %w", err)
	}

	return &u, nil
}

func (r *userPostgresRepository) CreateTx(ctx context.Context, tx pgx.Tx, email, hashedPassword, role string) (*model.User, error) {
	var u model.User
	err := tx.QueryRow(ctx,
		`INSERT INTO users (email, password, role, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, email, role, created_at, updated_at`,
		email, hashedPassword, role,
	).Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("menyimpan user baru: %w", err)
	}

	return &u, nil
}
