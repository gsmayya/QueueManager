package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"nodequeue-master-service/models"
	"nodequeue-master-service/utils"

	"github.com/google/uuid"
)

func (s *Service) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Capacity < 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "capacity must be >= 0")
		return
	}

	// Stable ID: generate if not provided.
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	out, err := s.roomStore.CreateRoom(r.Context(), req)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, out)
}

func (s *Service) ListRooms(w http.ResponseWriter, r *http.Request) {
	includeDeleted := true
	if v := r.URL.Query().Get("include_deleted"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			includeDeleted = b
		}
	}
	out, err := s.roomStore.ListRooms(r.Context(), includeDeleted)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) GetRoom(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	out, err := s.roomStore.GetRoom(r.Context(), id)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) UpdateRoom(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	var req models.UpdateRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		req.Name = &v
		if v == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
	}
	if req.Capacity != nil && *req.Capacity < 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "capacity must be >= 0")
		return
	}

	out, err := s.roomStore.UpdateRoom(r.Context(), id, req)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) DeleteRoom(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := s.roomStore.SoftDeleteRoom(r.Context(), id); err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
