package handlers

import "nodequeue-master-service/store"

type Service struct {
	store     store.Store
	roomStore store.RoomStore
}

func NewService(s store.Store, roomStore store.RoomStore) *Service {
	return &Service{store: s, roomStore: roomStore}
}
