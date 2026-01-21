package handlers

import (
	"net/http"
	"strings"

	"queue-common/logging"
	"queue-common/models"
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

	logging.Infof("[entities] create requested name=%q phone=%q", req.Name, maskPhone(req.Phone))
	out, err := s.entityStore.CreateEntity(r.Context(), req)
	if err != nil {
		logging.Errorf("[entities] create failed name=%q phone=%q err=%v", req.Name, maskPhone(req.Phone), err)
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.Infof("[entities] create succeeded id=%s name=%q", out.ID, out.Name)
	utils.RespondWithJSON(w, http.StatusCreated, out)
}

func (s *Service) ListEntities(w http.ResponseWriter, r *http.Request) {
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	var (
		out []models.Entity
		err error
	)
	if phone != "" {
		out, err = s.entityStore.ListEntitiesByPhone(r.Context(), phone)
	} else {
		out, err = s.entityStore.ListEntities(r.Context())
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
	out, err := s.entityStore.GetEntity(r.Context(), id)
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

	var nameVal any = nil
	if req.Name != nil {
		nameVal = *req.Name
	}
	var phoneVal any = nil
	if req.Phone != nil {
		phoneVal = maskPhone(*req.Phone)
	}
	logging.Infof("[entities] update requested id=%s name=%v phone=%v", id, nameVal, phoneVal)
	out, err := s.entityStore.UpdateEntity(r.Context(), id, req)
	if err != nil {
		logging.Errorf("[entities] update failed id=%s err=%v", id, err)
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.Infof("[entities] update succeeded id=%s", out.ID)
	utils.RespondWithJSON(w, http.StatusOK, out)
}

func (s *Service) DeleteEntity(w http.ResponseWriter, r *http.Request, id string) {
	if !utils.ParseUUIDParam(w, id) {
		return
	}
	if err := s.entityStore.DeleteEntity(r.Context(), id); err != nil {
		if mapStoreErr(w, err) {
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
