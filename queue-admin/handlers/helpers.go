package handlers

import (
	"errors"
	"net/http"

	"queue-common/store"
	"queue-common/utils"
)

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
