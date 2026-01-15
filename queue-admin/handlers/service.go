package handlers

import (
	"queue-common/store"
)

type Service struct {
	entityStore store.EntityStore
	userStore   store.UserStore
	roomStore   store.RoomStore
}

func NewService(entityStore store.EntityStore, userStore store.UserStore, roomStore store.RoomStore) *Service {
	return &Service{entityStore: entityStore, userStore: userStore, roomStore: roomStore}
}
