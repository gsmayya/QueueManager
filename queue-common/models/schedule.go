package models

import "time"

// Schedule defines a recurring job that periodically creates a node and assigns it to a resource.
// The time limit applies starting when the created node is added to the resource waiting queue.
type Schedule struct {
	ID               string    `json:"id"`
	EntityID         string    `json:"entity_id"`
	ResourceID       string    `json:"resource_id"`
	IntervalSeconds  int       `json:"interval_seconds"`
	TimeLimitSeconds int       `json:"time_limit_seconds"`
	WaitingExpirySeconds int   `json:"waiting_expiry_seconds"`
	EndsAt           *time.Time `json:"ends_at,omitempty"`
	Enabled          bool      `json:"enabled"`
	NextRunAt        time.Time `json:"next_run_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateScheduleRequest struct {
	EntityID         string `json:"entity_id"`
	ResourceID       string `json:"resource_id"`
	IntervalSeconds  int    `json:"interval_seconds"`
	TimeLimitSeconds int    `json:"time_limit_seconds"`
	WaitingExpirySeconds int `json:"waiting_expiry_seconds"`
	EndsAt           *time.Time `json:"ends_at,omitempty"`
	// Optional: if provided, overrides default next_run_at=now.
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	// Optional: default true.
	Enabled *bool `json:"enabled,omitempty"`
}

type UpdateScheduleRequest struct {
	ResourceID       *string    `json:"resource_id,omitempty"`
	IntervalSeconds  *int       `json:"interval_seconds,omitempty"`
	TimeLimitSeconds *int       `json:"time_limit_seconds,omitempty"`
	WaitingExpirySeconds *int   `json:"waiting_expiry_seconds,omitempty"`
	EndsAt           *time.Time `json:"ends_at,omitempty"`
	Enabled          *bool      `json:"enabled,omitempty"`
	NextRunAt        *time.Time `json:"next_run_at,omitempty"`
}

