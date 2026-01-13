package queueservice

import (
	"log"
	"net/http"
	"time"

	"nodequeue-service/db"
	"nodequeue-service/node"
	"nodequeue-service/utils"
)

// ResourcesMetricsHandler handles GET /resources/metrics.
// It returns per-resource metrics computed from node lifecycle logs.
//
// Best-effort persistence behavior:
// - If a DB Store is configured and reachable, we prefer DB node_logs (durable across restarts).
// - Otherwise we fall back to in-memory node logs (session-only).
func (qs *QueueService) ResourcesMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	now := time.Now()
	log.Printf("[API] GET /resources/metrics - Request")

	qs.mu.RLock()
	// Snapshot current resource queue sizes.
	resourceCounts := make(map[string][2]int, len(qs.resources))
	for rid, res := range qs.resources {
		waiting, service := res.QueueCounts()
		resourceCounts[rid] = [2]int{waiting, service}
	}

	// Snapshot node metadata + in-memory logs (fallback if DB logs are unavailable).
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
	sessionStart := qs.sessionStart
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

	// Build node events from DB logs when present; otherwise use in-memory logs.
	logsByNode := make(map[string][]nodeEvent, len(snaps))
	var earliestTS time.Time
	haveEarliest := false
	for id := range snaps {
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
		logsByNode[id] = evs

		// When DB logs are available, make sessionStart reflect the earliest available event timestamp
		// so the response metadata matches the data window.
		if dbLogs != nil {
			for _, ev := range evs {
				if !haveEarliest || ev.TS.Before(earliestTS) {
					earliestTS = ev.TS
					haveEarliest = true
				}
			}
		}
	}
	if haveEarliest {
		sessionStart = earliestTS
	}

	resp := computeResourcesSessionMetrics(sessionStart, now, resourceCounts, snaps, logsByNode)

	duration := time.Since(startTime)
	log.Printf("[API] GET /resources/metrics - SUCCESS: Returning %d resources (took %v)", len(resp.Resources), duration)
	utils.RespondWithJSON(w, http.StatusOK, resp)
}
