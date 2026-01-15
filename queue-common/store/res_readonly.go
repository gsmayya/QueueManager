package store

import (
	"context"
	"database/sql"
	"queue-common/models"
)

type ResStoreImpl struct {
	db *sql.DB
}

func NewResStore(db *sql.DB) *ResStoreImpl {
	return &ResStoreImpl{db: db}
}

func (s *ResStoreImpl) ListResources(ctx context.Context) ([]*models.Resource, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, capacity FROM resources WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*models.Resource, 0)
	for rows.Next() {
		var id string
		var name string
		var cap int
		if err := rows.Scan(&id, &name, &cap); err != nil {
			return nil, err
		}
		r := models.NewResource(id, cap)
		r.Name = name
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
