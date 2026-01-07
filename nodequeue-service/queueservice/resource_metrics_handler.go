package queueservice

import (
	"log"
	"net/http"
	"time"

	"nodequeue-service/node"
	"nodequeue-service/utils"
)

// ResourcesMetricsHandler handles GET /resources/metrics.
// It returns session-only (in-memory) per-resource metrics computed from node lifecycle logs.
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

	// Snapshot node metadata + in-memory logs.
	snaps := make(map[string]nodeSnapshot, len(qs.nodes))
	logsByNode := make(map[string][]nodeEvent, len(qs.nodes))
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

		if len(n.Log) > 0 {
			cp := make([]node.NodeLog, len(n.Log))
			copy(cp, n.Log)
			logsByNode[id] = toNodeEventsFromInMemory(cp)
		} else {
			logsByNode[id] = nil
		}
	}
	sessionStart := qs.sessionStart
	qs.mu.RUnlock()

	resp := computeResourcesSessionMetrics(sessionStart, now, resourceCounts, snaps, logsByNode)

	duration := time.Since(startTime)
	log.Printf("[API] GET /resources/metrics - SUCCESS: Returning %d resources (took %v)", len(resp.Resources), duration)
	utils.RespondWithJSON(w, http.StatusOK, resp)
}
