package handlers

import (
	"net/http"
	"strings"

	"queue-common/logging"
	"queue-common/models"
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

	// Never log plaintext passwords.
	logging.Infof("[users] create requested user_id=%q name=%q email=%q", req.UserID, req.Name, req.Email)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logging.Errorf("[users] create failed to hash password user_id=%q email=%q err=%v", req.UserID, req.Email, err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	out, err := s.userStore.CreateUser(r.Context(), req.UserID, req.Name, req.Email, string(hash))
	if err != nil {
		logging.Errorf("[users] create failed user_id=%q email=%q err=%v", req.UserID, req.Email, err)
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.Infof("[users] create succeeded id=%s user_id=%q", out.ID, out.UserID)
	utils.RespondWithJSON(w, http.StatusCreated, out)
}

func (s *Service) ListUsers(w http.ResponseWriter, r *http.Request) {
	out, err := s.userStore.ListUsers(r.Context())
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
	out, err := s.userStore.GetUser(r.Context(), id)
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
		// Never log plaintext passwords.
		logging.Infof("[users] update requested id=%s user_id=%v name=%v email=%v password_updated=true", id, req.UserID != nil, req.Name != nil, req.Email != nil)
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			logging.Errorf("[users] update failed to hash password id=%s err=%v", id, err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		h := string(hash)
		hashPtr = &h
	}
	if req.Password == nil {
		logging.Infof("[users] update requested id=%s user_id=%v name=%v email=%v password_updated=false", id, req.UserID != nil, req.Name != nil, req.Email != nil)
	}

	out, err := s.userStore.UpdateUser(r.Context(), id, req.UserID, req.Name, req.Email, hashPtr)
	if err != nil {
		logging.Errorf("[users] update failed id=%s err=%v", id, err)
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.Infof("[users] update succeeded id=%s", out.ID)
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	if err := s.userStore.DeleteUser(r.Context(), id); err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
