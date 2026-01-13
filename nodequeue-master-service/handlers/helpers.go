package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"nodequeue-master-service/store"
	"nodequeue-master-service/utils"

	"github.com/google/uuid"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

func parseUUIDParam(w http.ResponseWriter, id string) bool {
	if _, err := uuid.Parse(id); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return false
	}
	return true
}

func mapStoreErr(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrNotFound):
		utils.RespondWithError(w, http.StatusNotFound, "Not found")
		return true
	case errors.Is(err, store.ErrConflict):
		utils.RespondWithError(w, http.StatusConflict, "Conflict")
		return true
	default:
		return false
	}
}
