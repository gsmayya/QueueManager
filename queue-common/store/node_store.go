package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type NodeStoreImpl struct {
	db *sql.DB
}

func NewNodeStore(db *sql.DB) *NodeStoreImpl {
	return &NodeStoreImpl{db: db}
}

func (s *NodeStoreImpl) EnsureEntity(ctx context.Context, entityID, entityName string, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entities (id, name, created_at)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, entityID, entityName, createdAt)
	return err
}

func (s *NodeStoreImpl) ListNodes(ctx context.Context) ([]PersistedNode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			n.id::text,
			n.entity_id::text,
			e.name,
			COALESCE(n.node_name, ''),
			n.resource_id,
			n.schedule_id::text,
			n.time_limit_seconds,
			n.waiting_expiry_seconds,
			n.assigned_at,
			n.due_at,
			n.expires_at,
			n.delay_flag,
			n.expired,
			n.expired_at,
			n.completed,
			n.created_at
		FROM nodes n
		JOIN entities e ON e.id = n.entity_id
		WHERE n.completed = false
		ORDER BY n.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PersistedNode, 0)
	for rows.Next() {
		var pn PersistedNode
		var scheduleID sql.NullString
		var tls sql.NullInt64
		var wes sql.NullInt64
		var assignedAt sql.NullTime
		var dueAt sql.NullTime
		var expiresAt sql.NullTime
		var expiredAt sql.NullTime
		if err := rows.Scan(
			&pn.NodeID,
			&pn.EntityID,
			&pn.EntityName,
			&pn.NodeName,
			&pn.ResourceID,
			&scheduleID,
			&tls,
			&wes,
			&assignedAt,
			&dueAt,
			&expiresAt,
			&pn.DelayFlag,
			&pn.Expired,
			&expiredAt,
			&pn.Completed,
			&pn.CreatedAt,
		); err != nil {
			return nil, err
		}
		if scheduleID.Valid {
			v := scheduleID.String
			pn.ScheduleID = &v
		}
		if tls.Valid {
			v := int(tls.Int64)
			pn.TimeLimitSeconds = &v
		}
		if wes.Valid {
			v := int(wes.Int64)
			pn.WaitingExpirySeconds = &v
		}
		if assignedAt.Valid {
			v := assignedAt.Time
			pn.AssignedAt = &v
		}
		if dueAt.Valid {
			v := dueAt.Time
			pn.DueAt = &v
		}
		if expiresAt.Valid {
			v := expiresAt.Time
			pn.ExpiresAt = &v
		}
		if expiredAt.Valid {
			v := expiredAt.Time
			pn.ExpiredAt = &v
		}
		out = append(out, pn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NodeStoreImpl) ListLatestNodeStates(ctx context.Context) (map[string]NodeState, error) {
	// Latest service/waiting state per node based on node_logs.
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (node_id) node_id::text, action, ts
		FROM node_logs
		WHERE action IN ('moved_to_waiting_queue', 'moved_to_service_queue')
		ORDER BY node_id, ts DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]NodeState)
	for rows.Next() {
		var nodeID string
		var action string
		var ts time.Time
		if err := rows.Scan(&nodeID, &action, &ts); err != nil {
			return nil, err
		}
		kind := QueueKindWaiting
		if action == "moved_to_service_queue" {
			kind = QueueKindService
		}
		out[nodeID] = NodeState{Queue: kind, TS: ts}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NodeStoreImpl) ListNodeLogs(ctx context.Context, nodeIDs []string) (map[string][]NodeLogRow, error) {
	out := make(map[string][]NodeLogRow)
	if len(nodeIDs) == 0 {
		return out, nil
	}

	// Build a safe IN list: ($1::uuid, $2::uuid, ...)
	var b strings.Builder
	b.WriteString(`
		SELECT node_id::text, action, resource_id, ts
		FROM node_logs
		WHERE node_id IN (`)
	args := make([]any, 0, len(nodeIDs))
	for i, id := range nodeIDs {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("$%d::uuid", i+1))
		args = append(args, id)
	}
	b.WriteString(`)
		ORDER BY node_id, ts ASC
	`)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var nodeID string
		var action string
		var rid sql.NullString
		var ts time.Time
		if err := rows.Scan(&nodeID, &action, &rid, &ts); err != nil {
			return nil, err
		}
		var rp *string
		if rid.Valid {
			v := rid.String
			rp = &v
		}
		out[nodeID] = append(out[nodeID], NodeLogRow{
			NodeID:     nodeID,
			Action:     action,
			ResourceID: rp,
			TS:         ts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NodeStoreImpl) PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName, nodeName string, createdAt time.Time) error {
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
		`INSERT INTO nodes (id, entity_id, node_name, completed, created_at) VALUES ($1::uuid, $2::uuid, $3, false, $4)
		 ON CONFLICT (id) DO NOTHING`,
		nodeID, entityID, nodeName, createdAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *NodeStoreImpl) UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET resource_id = $2 WHERE id = $1::uuid`,
		nodeID, resourceID,
	)
	return err
}

func (s *NodeStoreImpl) MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET completed = $2, resource_id = CASE WHEN $2 THEN NULL ELSE resource_id END WHERE id = $1::uuid`,
		nodeID, completed,
	)
	return err
}

func (s *NodeStoreImpl) InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_logs (node_id, action, resource_id, ts) VALUES ($1::uuid, $2, $3, $4)`,
		nodeID, action, resourceID, ts,
	)
	return err
}

func (s *NodeStoreImpl) UpdateNodeScheduling(ctx context.Context, nodeID string, scheduleID *string, timeLimitSeconds *int, assignedAt, dueAt *time.Time, delayFlag *bool) error {
	var tl any = nil
	if timeLimitSeconds != nil {
		tl = *timeLimitSeconds
	}
	var as any = nil
	if assignedAt != nil {
		as = *assignedAt
	}
	var du any = nil
	if dueAt != nil {
		du = *dueAt
	}
	var df any = nil
	if delayFlag != nil {
		df = *delayFlag
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE nodes
		SET
			schedule_id = COALESCE($2::uuid, schedule_id),
			time_limit_seconds = COALESCE($3, time_limit_seconds),
			assigned_at = COALESCE($4, assigned_at),
			due_at = COALESCE($5, due_at),
			delay_flag = COALESCE($6, delay_flag)
		WHERE id = $1::uuid
	`, nodeID, scheduleID, tl, as, du, df)
	return err
}

func (s *NodeStoreImpl) UpdateNodeExpiry(ctx context.Context, nodeID string, waitingExpirySeconds *int, expiresAt *time.Time) error {
	var wes any = nil
	if waitingExpirySeconds != nil {
		wes = *waitingExpirySeconds
	}
	var ex any = nil
	if expiresAt != nil {
		ex = *expiresAt
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE nodes
		SET
			waiting_expiry_seconds = COALESCE($2, waiting_expiry_seconds),
			expires_at = COALESCE($3, expires_at)
		WHERE id = $1::uuid
	`, nodeID, wes, ex)
	return err
}

func (s *NodeStoreImpl) MarkNodeExpired(ctx context.Context, nodeID string, expiredAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE nodes
		SET
			completed = true,
			expired = true,
			expired_at = $2,
			resource_id = NULL
		WHERE id = $1::uuid
	`, nodeID, expiredAt)
	return err
}

func (s *NodeStoreImpl) HasActiveNodeForSchedule(ctx context.Context, scheduleID string) (bool, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM nodes
			WHERE schedule_id = $1::uuid AND completed = false
		)
	`, scheduleID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *NodeStoreImpl) MarkOverdueNodes(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE nodes
		SET delay_flag = true
		WHERE completed = false
		  AND delay_flag = false
		  AND due_at IS NOT NULL
		  AND due_at < $1
	`, now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
