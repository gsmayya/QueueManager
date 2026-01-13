package handlers

import (
	"net/http"
	"strings"

	"nodequeue-master-service/models"
	"queue-common/utils"
)

func (s *Service) CreateEntity(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEntityRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" || req.Phone == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "name and phone are required")
		return
	}

	out, err := s.store.CreateEntity(r.Context(), req)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, out)
}

func (s *Service) ListEntities(w http.ResponseWriter, r *http.Request) {
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	var (
		out []models.Entity
		err error
	)
	if phone != "" {
		out, err = s.store.ListEntitiesByPhone(r.Context(), phone)
	} else {
		out, err = s.store.ListEntities(r.Context())
	}
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) GetEntity(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	out, err := s.store.GetEntity(r.Context(), id)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) UpdateEntity(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	var req models.UpdateEntityRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	// If provided, enforce non-empty for required fields.
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		req.Name = &v
		if v == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
	}
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		req.Phone = &v
		if v == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "phone cannot be empty")
			return
		}
	}

	out, err := s.store.UpdateEntity(r.Context(), id, req)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) DeleteEntity(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	if err := s.store.DeleteEntity(r.Context(), id); err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
