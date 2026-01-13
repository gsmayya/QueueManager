package store

import (
	"context"

	"queue-admin/models"
)

type RoomStore interface {
	CreateRoom(ctx context.Context, in models.CreateRoomRequest) (models.Room, error)
	ListRooms(ctx context.Context, includeDeleted bool) ([]models.Room, error)
	GetRoom(ctx context.Context, id string) (models.Room, error)
	UpdateRoom(ctx context.Context, id string, in models.UpdateRoomRequest) (models.Room, error)
	SoftDeleteRoom(ctx context.Context, id string) error
}
