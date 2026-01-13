package handlers

import (
	"net/http"
	"strings"

	"queue-admin/models"
	"queue-common/utils"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.UserID == "" || req.Name == "" || req.Email == "" || strings.TrimSpace(req.Password) == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "user_id, name, email, password are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	out, err := s.store.CreateUser(r.Context(), req.UserID, req.Name, req.Email, string(hash))
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, out)
}

func (s *Service) ListUsers(w http.ResponseWriter, r *http.Request) {
	out, err := s.store.ListUsers(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) GetUser(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	out, err := s.store.GetUser(r.Context(), id)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	var req models.UpdateUserRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		return &v
	}

	req.UserID = trimPtr(req.UserID)
	req.Name = trimPtr(req.Name)
	req.Email = trimPtr(req.Email)
	if req.UserID != nil && *req.UserID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "user_id cannot be empty")
		return
	}
	if req.Name != nil && *req.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}
	if req.Email != nil && *req.Email == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email cannot be empty")
		return
	}

	var hashPtr *string
	if req.Password != nil {
		pw := strings.TrimSpace(*req.Password)
		if pw == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "password cannot be empty")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		h := string(hash)
		hashPtr = &h
	}

	out, err := s.store.UpdateUser(r.Context(), id, req.UserID, req.Name, req.Email, hashPtr)
	if err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	if err := s.store.DeleteUser(r.Context(), id); err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
