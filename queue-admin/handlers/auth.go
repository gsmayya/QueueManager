package handlers

import (
	"net/http"
	"strings"

	"queue-common/logging"
	"queue-common/models"
	"queue-common/store"
	"queue-common/utils"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	email := strings.TrimSpace(req.Email)
	pw := strings.TrimSpace(req.Password)
	if email == "" || pw == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	u, hash, err := s.userStore.GetUserAuthByEmail(r.Context(), email)
	if err != nil {
		// Do not leak whether an email exists.
		if err == store.ErrNotFound {
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		logging.Errorf("[auth] login failed email=%q err=%v", email, err)
		utils.RespondWithError(w, http.StatusInternalServerError, "login failed")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	logging.Infof("[auth] login succeeded user_id=%q email=%q", u.UserID, u.Email)
	utils.RespondWithJSON(w, http.StatusOK, u)
}
