package store

import (
	"errors"
	"time"
)

var (
	// ErrNotFound indicates the requested record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a uniqueness constraint conflict.
	ErrConflict = errors.New("conflict")
)

type PersistedNode struct {
	NodeID     string
	EntityID   string
	EntityName string
	NodeName   string
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
