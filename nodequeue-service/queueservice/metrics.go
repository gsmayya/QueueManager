package queueservice

import (
	"sort"
	"time"

	"nodequeue-service/db"
	"nodequeue-service/node"
)

// WaitingSegment represents time spent waiting in a given resource.
// It starts when the node is moved into that resource's waiting queue and ends when it is
// allocated into that resource's service queue (or when it is moved away / completed).
type WaitingSegment struct {
	ResourceID string    `json:"resource_id"`
	StartTS    time.Time `json:"start_ts"`
	EndTS      time.Time `json:"end_ts"`
	DurationMS int64     `json:"duration_ms"`
}

// NodeMetrics is a computed view over a node's lifecycle.
type NodeMetrics struct {
	ID                  string           `json:"id"`
	EntityName          string           `json:"entity_name"`
	CreatedAt           time.Time        `json:"created_at"`
	Completed           bool             `json:"completed"`
	TotalTimeInSystemMS int64            `json:"total_time_in_system_ms"`
	WaitingSegments     []WaitingSegment `json:"waiting_segments"`
}

// NodesMetricsResponse is the response payload for GET /nodes/metrics.
type NodesMetricsResponse struct {
	ActiveNodes    []NodeMetrics `json:"active_nodes"`
	CompletedNodes []NodeMetrics `json:"completed_nodes"`
}

type nodeEvent struct {
	Action     string
	ResourceID string
	TS         time.Time
}

type nodeSnapshot struct {
	ID        string
	Entity    string
	CreatedAt time.Time
	Completed bool
}

func toNodeEventsFromInMemory(logs []node.NodeLog) []nodeEvent {
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

func toNodeEventsFromDB(rows []db.NodeLogRow) []nodeEvent {
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

func computeNodeMetrics(now time.Time, n nodeSnapshot, events []nodeEvent) NodeMetrics {
	// Sort to make computation deterministic even if logs are appended out-of-order.
	sort.SliceStable(events, func(i, j int) bool { return events[i].TS.Before(events[j].TS) })

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
			segments = append(segments, WaitingSegment{
				ResourceID: ev.ResourceID,
				StartTS:    ev.TS,
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
		CreatedAt:           n.CreatedAt,
		Completed:           n.Completed,
		TotalTimeInSystemMS: total.Milliseconds(),
		WaitingSegments:     segments,
	}
}
