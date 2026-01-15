package store

import (
	"context"
	"database/sql"
	"errors"

	"queue-common/models"
)

type EntityStoreImpl struct {
	db *sql.DB
}

func NewEntityStore(db *sql.DB) *EntityStoreImpl {
	return &EntityStoreImpl{db: db}
}

// --- Entities

func (s *EntityStoreImpl) CreateEntity(ctx context.Context, in models.CreateEntityRequest) (models.Entity, error) {
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

func (s *EntityStoreImpl) ListEntities(ctx context.Context) ([]models.Entity, error) {
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

func (s *EntityStoreImpl) ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error) {
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

func (s *EntityStoreImpl) GetEntity(ctx context.Context, id string) (models.Entity, error) {
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

func (s *EntityStoreImpl) UpdateEntity(ctx context.Context, id string, in models.UpdateEntityRequest) (models.Entity, error) {
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

func (s *EntityStoreImpl) DeleteEntity(ctx context.Context, id string) error {
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
