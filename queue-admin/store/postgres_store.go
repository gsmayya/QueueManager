package store

import (
	"context"
	"database/sql"
	"errors"

	"queue-admin/models"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// --- Entities

func (s *PostgresStore) CreateEntity(ctx context.Context, in models.CreateEntityRequest) (models.Entity, error) {
	var out models.Entity
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO entities (name, phone, email)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, phone, email, joining_date
	`, in.Name, in.Phone, in.Email)
	if err := row.Scan(&out.ID, &out.Name, &out.Phone, &out.Email, &out.JoiningDate); err != nil {
		if isUniqueViolation(err) {
			return models.Entity{}, ErrConflict
		}
		return models.Entity{}, err
	}
	return out, nil
}

func (s *PostgresStore) ListEntities(ctx context.Context) ([]models.Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, name, phone, email, joining_date
		FROM entities
		ORDER BY joining_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Entity, 0)
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Phone, &e.Email, &e.JoiningDate); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, name, phone, email, joining_date
		FROM entities
		WHERE phone = $1
		ORDER BY joining_date DESC
	`, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Entity, 0)
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Phone, &e.Email, &e.JoiningDate); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetEntity(ctx context.Context, id string) (models.Entity, error) {
	var e models.Entity
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, name, phone, email, joining_date
		FROM entities
		WHERE id = $1::uuid
	`, id).Scan(&e.ID, &e.Name, &e.Phone, &e.Email, &e.JoiningDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Entity{}, ErrNotFound
		}
		return models.Entity{}, err
	}
	return e, nil
}

func (s *PostgresStore) UpdateEntity(ctx context.Context, id string, in models.UpdateEntityRequest) (models.Entity, error) {
	var e models.Entity
	err := s.db.QueryRowContext(ctx, `
		UPDATE entities
		SET
			name = COALESCE($2, name),
			phone = COALESCE($3, phone),
			email = COALESCE($4, email),
			joining_date = COALESCE($5, joining_date)
		WHERE id = $1::uuid
		RETURNING id::text, name, phone, email, joining_date
	`, id, in.Name, in.Phone, in.Email, in.JoiningDate).Scan(&e.ID, &e.Name, &e.Phone, &e.Email, &e.JoiningDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Entity{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return models.Entity{}, ErrConflict
		}
		return models.Entity{}, err
	}
	return e, nil
}

func (s *PostgresStore) DeleteEntity(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM entities WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Users

func (s *PostgresStore) CreateUser(ctx context.Context, userID, name, email, passwordHash string) (models.User, error) {
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

func (s *PostgresStore) ListUsers(ctx context.Context) ([]models.User, error) {
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

func (s *PostgresStore) GetUser(ctx context.Context, id string) (models.User, error) {
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

func (s *PostgresStore) UpdateUser(ctx context.Context, id string, userID, name, email *string, passwordHash *string) (models.User, error) {
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

func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
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
