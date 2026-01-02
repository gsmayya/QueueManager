package db

import (
	"context"
	"time"

	"nodequeue-service/resource"
)

type PersistedNode struct {
	NodeID     string
	EntityName string
	ResourceID *string
	Completed  bool
	CreatedAt  time.Time
}

type QueueKind string

const (
	QueueKindWaiting QueueKind = "waiting"
	QueueKindService QueueKind = "service"
)

type NodeState struct {
	Queue QueueKind
	TS    time.Time
}

// NodeLogRow is a persisted lifecycle/audit event for a node.
// It is intentionally stored in the db package to avoid coupling Store to the node package.
type NodeLogRow struct {
	NodeID     string
	Action     string
	ResourceID *string
	TS         time.Time
}

// Store is an optional persistence/audit sink for QueueService.
// Implementations should be safe for best-effort writes (callers may ignore errors to keep API behavior stable).
type Store interface {
	ListResources(ctx context.Context) ([]*resource.Resource, error)
	ListNodes(ctx context.Context) ([]PersistedNode, error)
	ListLatestNodeStates(ctx context.Context) (map[string]NodeState, error)
	ListNodeLogs(ctx context.Context, nodeIDs []string) (map[string][]NodeLogRow, error)

	PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName string, createdAt time.Time) error
	UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error
	MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error
	InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error
}
