package queueservice

import (
	"log"
	"net/http"
	"sort"
	"time"

	"nodequeue-service/db"
	"nodequeue-service/node"
	"nodequeue-service/utils"
)

// NodesMetricsHandler handles GET /nodes/metrics.
// It returns all nodes (active + completed) along with computed time-in-system and waiting segments.
func (qs *QueueService) NodesMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	now := time.Now()
	log.Printf("[API] GET /nodes/metrics - Request")

	qs.mu.RLock()
	nodeIDs := make([]string, 0, len(qs.nodes))
	snaps := make(map[string]nodeSnapshot, len(qs.nodes))
	memLogs := make(map[string][]node.NodeLog, len(qs.nodes))
	for id, n := range qs.nodes {
		entityName := ""
		if n.Entity != nil {
			entityName = n.Entity.Name
		}
		snaps[id] = nodeSnapshot{
			ID:        n.ID,
			Entity:    entityName,
			CreatedAt: n.CreatedAt,
			Completed: n.Completed,
		}
		nodeIDs = append(nodeIDs, id)

		if len(n.Log) > 0 {
			cp := make([]node.NodeLog, len(n.Log))
			copy(cp, n.Log)
			memLogs[id] = cp
		} else {
			memLogs[id] = nil
		}
	}
	qs.mu.RUnlock()

	// Best-effort: prefer DB logs (complete history across restarts), fall back to in-memory logs.
	var dbLogs map[string][]db.NodeLogRow
	if qs.store != nil && len(nodeIDs) > 0 {
		var err error
		dbLogs, err = qs.store.ListNodeLogs(r.Context(), nodeIDs)
		if err != nil {
			log.Printf("[DB] ListNodeLogs failed (falling back to in-memory logs): %v", err)
			dbLogs = nil
		}
	}

	active := make([]NodeMetrics, 0)
	completed := make([]NodeMetrics, 0)
	for id, snap := range snaps {
		var evs []nodeEvent
		if dbLogs != nil {
			if rows := dbLogs[id]; len(rows) > 0 {
				evs = toNodeEventsFromDB(rows)
			} else {
				evs = toNodeEventsFromInMemory(memLogs[id])
			}
		} else {
			evs = toNodeEventsFromInMemory(memLogs[id])
		}

		m := computeNodeMetrics(now, snap, evs)
		if snap.Completed {
			completed = append(completed, m)
		} else {
			active = append(active, m)
		}
	}

	// Stable output ordering.
	sort.SliceStable(active, func(i, j int) bool { return active[i].CreatedAt.Before(active[j].CreatedAt) })
	sort.SliceStable(completed, func(i, j int) bool { return completed[i].CreatedAt.Before(completed[j].CreatedAt) })

	resp := NodesMetricsResponse{
		ActiveNodes:    active,
		CompletedNodes: completed,
	}

	duration := time.Since(startTime)
	log.Printf("[API] GET /nodes/metrics - SUCCESS: Returning %d active, %d completed (took %v)", len(active), len(completed), duration)
	utils.RespondWithJSON(w, http.StatusOK, resp)
}
