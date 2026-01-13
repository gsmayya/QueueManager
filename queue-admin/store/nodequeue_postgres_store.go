package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"queue-admin/models"

	"github.com/jackc/pgx/v5/pgconn"
)

type NodequeuePostgresStore struct {
	db *sql.DB
}

func NewNodequeuePostgresStore(db *sql.DB) *NodequeuePostgresStore {
	return &NodequeuePostgresStore{db: db}
}

func isUniqueViolationNodequeue(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (s *NodequeuePostgresStore) CreateRoom(ctx context.Context, in models.CreateRoomRequest) (models.Room, error) {
	var out models.Room
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO resources (id, name, capacity)
		VALUES ($1, $2, $3)
		RETURNING id, name, capacity, deleted_at, created_at
	`, in.ID, in.Name, in.Capacity)
	if err := row.Scan(&out.ID, &out.Name, &out.Capacity, &out.DeletedAt, &out.CreatedAt); err != nil {
		if isUniqueViolationNodequeue(err) {
			return models.Room{}, ErrConflict
		}
		return models.Room{}, err
	}
	return out, nil
}

func (s *NodequeuePostgresStore) ListRooms(ctx context.Context, includeDeleted bool) ([]models.Room, error) {
	q := `
		SELECT id, name, capacity, deleted_at, created_at
		FROM resources
	`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Room, 0)
	for rows.Next() {
		var r models.Room
		var deletedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.Capacity, &deletedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			r.DeletedAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *NodequeuePostgresStore) GetRoom(ctx context.Context, id string) (models.Room, error) {
	var r models.Room
	var deletedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, capacity, deleted_at, created_at
		FROM resources
		WHERE id = $1
	`, id).Scan(&r.ID, &r.Name, &r.Capacity, &deletedAt, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Room{}, ErrNotFound
		}
		return models.Room{}, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		r.DeletedAt = &t
	}
	return r, nil
}

func (s *NodequeuePostgresStore) UpdateRoom(ctx context.Context, id string, in models.UpdateRoomRequest) (models.Room, error) {
	// If DeletedAt is set explicitly, allow setting it (including null via omitted).
	var r models.Room
	var deletedAt sql.NullTime

	var deletedAtVal any = nil
	if in.DeletedAt != nil {
		deletedAtVal = *in.DeletedAt
	}

	err := s.db.QueryRowContext(ctx, `
		UPDATE resources
		SET
			name = COALESCE($2, name),
			capacity = COALESCE($3, capacity),
			deleted_at = COALESCE($4, deleted_at)
		WHERE id = $1
		RETURNING id, name, capacity, deleted_at, created_at
	`, id, in.Name, in.Capacity, deletedAtVal).Scan(&r.ID, &r.Name, &r.Capacity, &deletedAt, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Room{}, ErrNotFound
		}
		if isUniqueViolationNodequeue(err) {
			return models.Room{}, ErrConflict
		}
		return models.Room{}, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		r.DeletedAt = &t
	}
	return r, nil
}

func (s *NodequeuePostgresStore) SoftDeleteRoom(ctx context.Context, id string) error {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
		UPDATE resources SET deleted_at = $2 WHERE id = $1
	`, id, now)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
