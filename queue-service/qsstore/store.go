package qsstore

import (
	"context"
	"queue-common/models"
	. "queue-common/store"
	"time"
)

// Store is an optional persistence/audit sink for QueueService.
// Implementations should be safe for best-effort writes (callers may ignore errors to keep API behavior stable).
type Store interface {
	ListResources(ctx context.Context) ([]*models.Resource, error)
	ListNodes(ctx context.Context) ([]PersistedNode, error)
	ListLatestNodeStates(ctx context.Context) (map[string]NodeState, error)
	ListNodeLogs(ctx context.Context, nodeIDs []string) (map[string][]NodeLogRow, error)

	PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName, nodeName string, createdAt time.Time) error
	UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error
	MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error
	InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error
}
