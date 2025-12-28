package db

import (
	"context"
	"database/sql"
	"time"

	"nodequeue-service/resource"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) ListResources(ctx context.Context) ([]*resource.Resource, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, capacity FROM resources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*resource.Resource, 0)
	for rows.Next() {
		var id string
		var cap int
		if err := rows.Scan(&id, &cap); err != nil {
			return nil, err
		}
		out = append(out, resource.NewResource(id, cap))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName string, createdAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO entities (id, name, created_at) VALUES ($1::uuid, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		entityID, entityName, createdAt,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO nodes (id, entity_id, completed, created_at) VALUES ($1::uuid, $2::uuid, false, $3)
		 ON CONFLICT (id) DO NOTHING`,
		nodeID, entityID, createdAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET resource_id = $2 WHERE id = $1::uuid`,
		nodeID, resourceID,
	)
	return err
}

func (s *PostgresStore) MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET completed = $2, resource_id = CASE WHEN $2 THEN NULL ELSE resource_id END WHERE id = $1::uuid`,
		nodeID, completed,
	)
	return err
}

func (s *PostgresStore) InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_logs (node_id, action, resource_id, ts) VALUES ($1::uuid, $2, $3, $4)`,
		nodeID, action, resourceID, ts,
	)
	return err
}
