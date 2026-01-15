package store

import (
	"context"
	"errors"
	"queue-common/models"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

type EntityStore interface {
	// Entities
	CreateEntity(ctx context.Context, in models.CreateEntityRequest) (models.Entity, error)
	ListEntities(ctx context.Context) ([]models.Entity, error)
	ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error)
	GetEntity(ctx context.Context, id string) (models.Entity, error)
	UpdateEntity(ctx context.Context, id string, in models.UpdateEntityRequest) (models.Entity, error)
	DeleteEntity(ctx context.Context, id string) error
}

type UserStore interface {

	// Users (passwordHash is stored; plaintext must never be persisted).
	CreateUser(ctx context.Context, userID, name, email, passwordHash string) (models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	GetUser(ctx context.Context, id string) (models.User, error)
	UpdateUser(ctx context.Context, id string, userID, name, email *string, passwordHash *string) (models.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type RoomStore interface {
	CreateRoom(ctx context.Context, in models.CreateRoomRequest) (models.Room, error)
	ListRooms(ctx context.Context, includeDeleted bool) ([]models.Room, error)
	GetRoom(ctx context.Context, id string) (models.Room, error)
	UpdateRoom(ctx context.Context, id string, in models.UpdateRoomRequest) (models.Room, error)
	SoftDeleteRoom(ctx context.Context, id string) error
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
