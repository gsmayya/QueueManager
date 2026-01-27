package queueservice

import (
	"net/http"
	"time"

	"queue-common/logging"
	"queue-common/utils"
)

// SchedulesMetricsHandler handles GET /schedules/metrics.
func (qs *QueueService) SchedulesMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if qs.schedstore == nil {
		utils.RespondWithError(w, http.StatusServiceUnavailable, "scheduling persistence is not configured")
		return
	}
	now := time.Now()
	resp, err := qs.schedstore.GetSchedulesMetrics(r.Context(), now)
	if err != nil {
		logging.Errorf("GET /schedules/metrics failed err=%v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, resp)
}

