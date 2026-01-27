package models

import "time"

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
	ResourceID       string `json:"resource_id"`
	ResourceName     string `json:"resource_name"`
	ResourceCapacity int    `json:"resource_capacity"`

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

// WaitingSegment represents time spent waiting in a given resource.
// It starts when the node is moved into that resource's waiting queue and ends when it is
// allocated into that resource's service queue (or when it is moved away / completed).
type WaitingSegment struct {
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	StartTS      time.Time `json:"start_ts"`
	EndTS        time.Time `json:"end_ts"`
	DurationMS   int64     `json:"duration_ms"`
}

// NodeMetrics is a computed view over a node's lifecycle.
type NodeMetrics struct {
	ID                  string           `json:"id"`
	EntityName          string           `json:"entity_name"`
	NodeName            string           `json:"node_name,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	ScheduleID          *string          `json:"schedule_id,omitempty"`
	AssignedAt          *time.Time       `json:"assigned_at,omitempty"`
	DueAt               *time.Time       `json:"due_at,omitempty"`
	DelayFlag           bool             `json:"delay_flag"`
	Completed           bool             `json:"completed"`
	TotalTimeInSystemMS int64            `json:"total_time_in_system_ms"`
	WaitingSegments     []WaitingSegment `json:"waiting_segments"`
}

// NodesMetricsResponse is the response payload for GET /nodes/metrics.
type NodesMetricsResponse struct {
	ActiveNodes    []NodeMetrics `json:"active_nodes"`
	CompletedNodes []NodeMetrics `json:"completed_nodes"`
}

// --- Schedule metrics (DB-backed, computed from nodes + node_logs)

type SchedulesMetricsTotals struct {
	Now time.Time `json:"now"`

	TotalSchedules   int64 `json:"total_schedules"`
	EnabledSchedules int64 `json:"enabled_schedules"`
	EndedSchedules   int64 `json:"ended_schedules"`

	// Node outcomes (scheduled nodes only).
	FiredCount   int64 `json:"fired_count"`
	CompletedCount int64 `json:"completed_count"`
	ExpiredCount int64 `json:"expired_count"`

	CompletedWithinTimeLimitCount int64 `json:"completed_within_time_limit_count"`

	AvgAssignedToAllocateMS *int64 `json:"avg_assigned_to_allocate_ms,omitempty"`
	AvgAssignedToCompleteMS *int64 `json:"avg_assigned_to_complete_ms,omitempty"`
	AvgAssignedToExpiredMS  *int64 `json:"avg_assigned_to_expired_ms,omitempty"`
}

type ScheduleMetrics struct {
	ScheduleID          string     `json:"schedule_id"`
	EntityID            string     `json:"entity_id"`
	ResourceID          string     `json:"resource_id"`
	IntervalSeconds     int        `json:"interval_seconds"`
	TimeLimitSeconds    int        `json:"time_limit_seconds"`
	WaitingExpirySeconds int       `json:"waiting_expiry_seconds"`
	EndsAt              *time.Time `json:"ends_at,omitempty"`
	Enabled             bool       `json:"enabled"`
	NextRunAt           time.Time  `json:"next_run_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	FiredCount   int64 `json:"fired_count"`
	CompletedCount int64 `json:"completed_count"`
	ExpiredCount int64 `json:"expired_count"`

	CompletedWithinTimeLimitCount int64 `json:"completed_within_time_limit_count"`

	AvgAssignedToAllocateMS *int64 `json:"avg_assigned_to_allocate_ms,omitempty"`
	AvgAssignedToCompleteMS *int64 `json:"avg_assigned_to_complete_ms,omitempty"`
	AvgAssignedToExpiredMS  *int64 `json:"avg_assigned_to_expired_ms,omitempty"`
}

type SchedulesMetricsResponse struct {
	Totals    SchedulesMetricsTotals `json:"totals"`
	Schedules []ScheduleMetrics      `json:"schedules"`
}
