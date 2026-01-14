package store

import (
	"context"
	"errors"

	"queue-common/models"
)

var (
	// ErrNotFound indicates the requested record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a uniqueness constraint conflict.
	ErrConflict = errors.New("conflict")
)

type Store interface {
	// Entities
	CreateEntity(ctx context.Context, in models.CreateEntityRequest) (models.Entity, error)
	ListEntities(ctx context.Context) ([]models.Entity, error)
	ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error)
	GetEntity(ctx context.Context, id string) (models.Entity, error)
	UpdateEntity(ctx context.Context, id string, in models.UpdateEntityRequest) (models.Entity, error)
	DeleteEntity(ctx context.Context, id string) error

	// Users (passwordHash is stored; plaintext must never be persisted).
	CreateUser(ctx context.Context, userID, name, email, passwordHash string) (models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	GetUser(ctx context.Context, id string) (models.User, error)
	UpdateUser(ctx context.Context, id string, userID, name, email *string, passwordHash *string) (models.User, error)
	DeleteUser(ctx context.Context, id string) error
}
