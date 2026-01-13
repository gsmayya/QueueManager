package queueservice

import (
	"slices"
	"time"
)

// ServiceSegment represents time spent allocated (in service) for a given resource.
// It starts when the node is moved into that resource's service queue and ends when the node
// moves away to any waiting queue or is completed (or when closed at "now").
type ServiceSegment struct {
	ResourceID string    `json:"resource_id"`
	StartTS    time.Time `json:"start_ts"`
	EndTS      time.Time `json:"end_ts"`
	DurationMS int64     `json:"duration_ms"`
}

// ResourceSessionMetrics represents session-only (in-memory) metrics for a single resource.
type ResourceSessionMetrics struct {
	ResourceID string `json:"resource_id"`

	// TotalAdded counts how many times nodes were moved/assigned into this resource's waiting queue.
	// This counts revisits (i.e., total assignments), not unique node IDs.
	TotalAdded int64 `json:"total_added"`

	// TotalAllocated counts how many times nodes were promoted into this resource's service queue.
	TotalAllocated int64 `json:"total_allocated"`

	// Current queue sizes (point-in-time).
	CurrentWaiting   int `json:"current_waiting"`
	CurrentAllocated int `json:"current_allocated"`

	// Waiting segments aggregation.
	WaitingSegmentsCount int64 `json:"waiting_segments_count"`
	WaitingTotalMS       int64 `json:"waiting_total_ms"`
	AvgWaitingTimeMS     int64 `json:"avg_waiting_time_ms"`

	// Service segments aggregation.
	ServiceSegmentsCount int64 `json:"service_segments_count"`
	ServiceTotalMS       int64 `json:"service_total_ms"`
	AvgServiceTimeMS     int64 `json:"avg_service_time_ms"`
}

// ResourcesSessionMetricsResponse is the response payload for GET /resources/metrics.
type ResourcesSessionMetricsResponse struct {
	SessionStart time.Time                `json:"session_start"`
	Now          time.Time                `json:"now"`
	Resources    []ResourceSessionMetrics `json:"resources"`
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
