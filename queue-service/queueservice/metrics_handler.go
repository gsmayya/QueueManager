package queueservice

import (
	"net/http"
	"slices"
	"time"

	"queue-common/logging"
	. "queue-common/models"
	"queue-common/store"
	"queue-common/utils"
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
	logging.Debugf("GET /nodes/metrics request")

	qs.mu.RLock()
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
			NodeName:  n.NodeName,
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
	qs.mu.RUnlock()

	// Best-effort: prefer DB logs (complete history across restarts), fall back to in-memory logs.
	var dbLogs map[string][]store.NodeLogRow
	if qs.nodestore != nil && len(nodeIDs) > 0 {
		var err error
		dbLogs, err = qs.nodestore.ListNodeLogs(r.Context(), nodeIDs)
		if err != nil {
			logging.Debugf("[DB] ListNodeLogs failed (falling back to in-memory logs): %v", err)
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

		m := qs.computeNodeMetrics(now, snap, evs)
		if snap.Completed {
			completed = append(completed, m)
		} else {
			active = append(active, m)
		}
	}

	// Stable output ordering.
	slices.SortStableFunc(active, func(a, b NodeMetrics) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if b.CreatedAt.Before(a.CreatedAt) {
			return 1
		}
		return 0
	})
	slices.SortStableFunc(completed, func(a, b NodeMetrics) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if b.CreatedAt.Before(a.CreatedAt) {
			return 1
		}
		return 0
	})

	resp := NodesMetricsResponse{
		ActiveNodes:    active,
		CompletedNodes: completed,
	}

	duration := time.Since(startTime)
	logging.Debugf("GET /nodes/metrics success active=%d completed=%d took=%v", len(active), len(completed), duration)
	utils.RespondWithJSON(w, http.StatusOK, resp)
}

type nodeEvent struct {
	Action     string
	ResourceID string
	TS         time.Time
}

type nodeSnapshot struct {
	ID        string
	NodeName  string
	Entity    string
	CreatedAt time.Time
	Completed bool
}

func toNodeEventsFromInMemory(logs []NodeLog) []nodeEvent {
	out := make([]nodeEvent, 0, len(logs))
	for _, l := range logs {
		out = append(out, nodeEvent{
			Action:     l.Action,
			ResourceID: l.ResourceID,
			TS:         l.Timestamp,
		})
	}
	return out
}

func toNodeEventsFromDB(rows []store.NodeLogRow) []nodeEvent {
	out := make([]nodeEvent, 0, len(rows))
	for _, r := range rows {
		rid := ""
		if r.ResourceID != nil {
			rid = *r.ResourceID
		}
		out = append(out, nodeEvent{
			Action:     r.Action,
			ResourceID: rid,
			TS:         r.TS,
		})
	}
	return out
}

func (qs *QueueService) computeNodeMetrics(now time.Time, n nodeSnapshot, events []nodeEvent) NodeMetrics {
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

	segments := make([]WaitingSegment, 0)
	openIdx := -1
	var completedTS *time.Time

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
		case "moved_to_waiting_queue":
			// If we were already waiting somewhere, treat this as leaving that wait state.
			closeOpen(ev.TS)
			res, err := qs.GetResource(ev.ResourceID)
			if err != nil {
				logging.Debugf("[DB] GetResource failed: %v", err)
				continue
			}
			segments = append(segments, WaitingSegment{
				ResourceID:   ev.ResourceID,
				ResourceName: res.Name,
				StartTS:      ev.TS,
			})
			openIdx = len(segments) - 1

		case "moved_to_service_queue":
			// Only close if it matches the currently open wait segment.
			if openIdx != -1 && segments[openIdx].ResourceID == ev.ResourceID {
				closeOpen(ev.TS)
			}

		case "completed":
			// Freeze totals at completion time; also stop any ongoing waiting.
			ts := ev.TS
			completedTS = &ts
			closeOpen(ev.TS)
		}
	}

	// If still waiting, close at now.
	closeOpen(now)

	total := now.Sub(n.CreatedAt)
	if completedTS != nil {
		total = completedTS.Sub(n.CreatedAt)
	}
	if total < 0 {
		total = 0
	}

	return NodeMetrics{
		ID:                  n.ID,
		EntityName:          n.Entity,
		NodeName:            n.NodeName,
		CreatedAt:           n.CreatedAt,
		Completed:           n.Completed,
		TotalTimeInSystemMS: total.Milliseconds(),
		WaitingSegments:     segments,
	}
}
