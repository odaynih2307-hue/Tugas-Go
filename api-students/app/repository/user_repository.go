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

func (r *UserRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error) {
	where := " WHERE 1 = 1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND username ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung user: %w", err)
	}

	orderDir := "ASC"
	if q.Order == "desc" {
		orderDir = "DESC"
	}

	sortCol := "id"
	if q.Sort == "username" || q.Sort == "created_at" || q.Sort == "role" {
		sortCol = q.Sort
	}

	sqlText := fmt.Sprintf(
		`SELECT id, username, password, role, created_at
		 FROM users%s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		where, sortCol, orderDir, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) Update(
	ctx context.Context,
	u model.User,
) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(
		ctx,
		`UPDATE users SET username = $1
		 WHERE id = $2
		 RETURNING id, username, password, role, created_at`,
		u.Username, u.ID,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("memperbarui user: %w", err)
	}

	return &user, nil
}

// UpdateRole sengaja dipisah dari Update. Mengubah role adalah tindakan
// istimewa yang dijaga permission tersendiri.
func (r *UserRepository) UpdateRole(
	ctx context.Context,
	id int,
	role string,
) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(
		ctx,
		`UPDATE users SET role = $1
		 WHERE id = $2
		 RETURNING id, username, password, role, created_at`,
		role, id,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("mengubah role user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
