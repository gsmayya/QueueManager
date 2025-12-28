package db

import (
	"context"
	"time"

	"nodequeue-service/resource"
)

// Store is an optional persistence/audit sink for QueueService.
// Implementations should be safe for best-effort writes (callers may ignore errors to keep API behavior stable).
type Store interface {
	ListResources(ctx context.Context) ([]*resource.Resource, error)

	PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName string, createdAt time.Time) error
	UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error
	MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error
	InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error
}
