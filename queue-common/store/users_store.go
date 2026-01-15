package store

import (
	"context"
	"database/sql"
	"errors"

	"queue-common/models"
)

type UserStoreImpl struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStoreImpl {
	return &UserStoreImpl{db: db}
}

// --- Users

func (s *UserStoreImpl) CreateUser(ctx context.Context, userID, name, email, passwordHash string) (models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (user_id, name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, user_id, name, email, created_at
	`, userID, name, email, passwordHash).Scan(&u.ID, &u.UserID, &u.Name, &u.Email, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrConflict
		}
		return models.User{}, err
	}
	return u, nil
}

func (s *UserStoreImpl) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, user_id, name, email, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.UserID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *UserStoreImpl) GetUser(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, user_id, name, email, created_at
		FROM users
		WHERE id = $1::uuid
	`, id).Scan(&u.ID, &u.UserID, &u.Name, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, err
	}
	return u, nil
}

func (s *UserStoreImpl) UpdateUser(ctx context.Context, id string, userID, name, email *string, passwordHash *string) (models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx, `
		UPDATE users
		SET
			user_id = COALESCE($2, user_id),
			name = COALESCE($3, name),
			email = COALESCE($4, email),
			password_hash = COALESCE($5, password_hash)
		WHERE id = $1::uuid
		RETURNING id::text, user_id, name, email, created_at
	`, id, userID, name, email, passwordHash).Scan(&u.ID, &u.UserID, &u.Name, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return models.User{}, ErrConflict
		}
		return models.User{}, err
	}
	return u, nil
}

func (s *UserStoreImpl) DeleteUser(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
