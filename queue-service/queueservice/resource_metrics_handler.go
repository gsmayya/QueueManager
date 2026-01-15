package queueservice

import (
	"log"
	"net/http"
	"slices"
	"time"

	. "queue-common/models"
	"queue-common/store"
	"queue-common/utils"
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
	memLogs := make(map[string][]NodeLog, len(qs.nodes))
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
			cp := make([]NodeLog, len(n.Log))
			copy(cp, n.Log)
			memLogs[id] = cp
		} else {
			memLogs[id] = nil
		}
	}
	sessionStart := qs.sessionStart
	qs.mu.RUnlock()

	// Best-effort: prefer DB logs (complete history across restarts), fall back to in-memory logs.
	var dbLogs map[string][]store.NodeLogRow
	if qs.nodestore != nil && len(nodeIDs) > 0 {
		var err error
		dbLogs, err = qs.nodestore.ListNodeLogs(r.Context(), nodeIDs)
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

func computeServiceSegments(now time.Time, events []nodeEvent) []ServiceSegment {
	// Sort to make computation deterministic even if logs are appended out-of-order.
	slices.SortStableFunc(events, func(a, b nodeEvent) int {
		if a.TS.Before(b.TS) {
			return -1
		}
		if b.TS.Before(a.TS) {
			return 1
		}
		return 0
	})

	segments := make([]ServiceSegment, 0)
	openIdx := -1

	closeOpen := func(end time.Time) {
		if openIdx == -1 {
			return
		}
		segments[openIdx].EndTS = end
		d := end.Sub(segments[openIdx].StartTS)
		if d < 0 {
			d = 0
		}
		segments[openIdx].DurationMS = d.Milliseconds()
		openIdx = -1
	}

	for _, ev := range events {
		switch ev.Action {
		case "moved_to_service_queue":
			// Close any existing service segment (shouldn't normally happen).
			closeOpen(ev.TS)
			segments = append(segments, ServiceSegment{
				ResourceID: ev.ResourceID,
				StartTS:    ev.TS,
			})
			openIdx = len(segments) - 1

		case "moved_to_waiting_queue":
			// Moving to any waiting queue means leaving service (if currently in service).
			closeOpen(ev.TS)

		case "completed":
			closeOpen(ev.TS)
		}
	}

	// If still in service, close at now.
	closeOpen(now)

	return segments
}

type resourceAgg struct {
	m ResourceSessionMetrics
}

// computeResourcesSessionMetrics aggregates session-only metrics per resource from in-memory node logs.
//
// Semantics:
// - TotalAdded: counts moved_to_waiting_queue events per resource (counts revisits)
// - TotalAllocated: counts moved_to_service_queue events per resource
// - AvgWaitingTimeMS: average over waiting segments for that resource (open segments closed at now)
// - AvgServiceTimeMS: average over service segments for that resource (open segments closed at now)
func computeResourcesSessionMetrics(
	sessionStart time.Time,
	now time.Time,
	resourceCounts map[string][2]int, // rid -> [waiting, allocated]
	nodes map[string]nodeSnapshot,
	logsByNode map[string][]nodeEvent,
) ResourcesSessionMetricsResponse {
	aggs := make(map[string]*resourceAgg, len(resourceCounts))

	ensure := func(rid string) *resourceAgg {
		if a, ok := aggs[rid]; ok {
			return a
		}
		a := &resourceAgg{
			m: ResourceSessionMetrics{
				ResourceID: rid,
			},
		}
		if c, ok := resourceCounts[rid]; ok {
			a.m.CurrentWaiting = c[0]
			a.m.CurrentAllocated = c[1]
		}
		aggs[rid] = a
		return a
	}

	// Ensure all known resources exist in output even with zero activity.
	for rid := range resourceCounts {
		ensure(rid)
	}

	for nid := range nodes {
		evs := logsByNode[nid]
		if len(evs) == 0 {
			continue
		}

		// Count adds/allocations by resource based on raw events.
		for _, ev := range evs {
			switch ev.Action {
			case "moved_to_waiting_queue":
				if ev.ResourceID == "" {
					continue
				}
				ensure(ev.ResourceID).m.TotalAdded++
			case "moved_to_service_queue":
				if ev.ResourceID == "" {
					continue
				}
				ensure(ev.ResourceID).m.TotalAllocated++
			}
		}

		// Aggregate waiting durations from computed waiting segments.
		waiting := computeNodeMetrics(now, nodes[nid], evs).WaitingSegments
		for _, seg := range waiting {
			if seg.ResourceID == "" {
				continue
			}
			a := ensure(seg.ResourceID)
			a.m.WaitingSegmentsCount++
			a.m.WaitingTotalMS += seg.DurationMS
		}

		// Aggregate service durations from computed service segments.
		service := computeServiceSegments(now, evs)
		for _, seg := range service {
			if seg.ResourceID == "" {
				continue
			}
			a := ensure(seg.ResourceID)
			a.m.ServiceSegmentsCount++
			a.m.ServiceTotalMS += seg.DurationMS
		}
	}

	// Compute averages and return stable sorted list.
	out := make([]ResourceSessionMetrics, 0, len(aggs))
	for _, a := range aggs {
		if a.m.WaitingSegmentsCount > 0 {
			a.m.AvgWaitingTimeMS = a.m.WaitingTotalMS / a.m.WaitingSegmentsCount
		}
		if a.m.ServiceSegmentsCount > 0 {
			a.m.AvgServiceTimeMS = a.m.ServiceTotalMS / a.m.ServiceSegmentsCount
		}
		out = append(out, a.m)
	}

	slices.SortStableFunc(out, func(a, b ResourceSessionMetrics) int {
		if a.ResourceID < b.ResourceID {
			return -1
		}
		if b.ResourceID < a.ResourceID {
			return 1
		}
		return 0
	})

	return ResourcesSessionMetricsResponse{
		SessionStart: sessionStart,
		Now:          now,
		Resources:    out,
	}
}
