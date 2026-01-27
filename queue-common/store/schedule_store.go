package store

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"queue-common/models"
)

type ScheduleStoreImpl struct {
	db *sql.DB
}

func NewScheduleStore(db *sql.DB) *ScheduleStoreImpl {
	return &ScheduleStoreImpl{db: db}
}

func (s *ScheduleStoreImpl) CreateSchedule(ctx context.Context, in models.CreateScheduleRequest) (models.Schedule, error) {
	var out models.Schedule

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	nextRunAt := time.Now()
	if in.NextRunAt != nil {
		nextRunAt = *in.NextRunAt
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO schedules (entity_id, resource_id, interval_seconds, time_limit_seconds, waiting_expiry_seconds, ends_at, enabled, next_run_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING
			id::text,
			entity_id::text,
			resource_id,
			interval_seconds,
			time_limit_seconds,
			waiting_expiry_seconds,
			ends_at,
			enabled,
			next_run_at,
			created_at,
			updated_at
	`, in.EntityID, in.ResourceID, in.IntervalSeconds, in.TimeLimitSeconds, in.WaitingExpirySeconds, in.EndsAt, enabled, nextRunAt).Scan(
		&out.ID,
		&out.EntityID,
		&out.ResourceID,
		&out.IntervalSeconds,
		&out.TimeLimitSeconds,
		&out.WaitingExpirySeconds,
		&out.EndsAt,
		&out.Enabled,
		&out.NextRunAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Schedule{}, ErrConflict
		}
		return models.Schedule{}, err
	}
	return out, nil
}

func (s *ScheduleStoreImpl) ListSchedules(ctx context.Context) ([]models.Schedule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id::text,
			entity_id::text,
			resource_id,
			interval_seconds,
			time_limit_seconds,
			waiting_expiry_seconds,
			ends_at,
			enabled,
			next_run_at,
			created_at,
			updated_at
		FROM schedules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Schedule, 0)
	for rows.Next() {
		var sc models.Schedule
		if err := rows.Scan(
			&sc.ID,
			&sc.EntityID,
			&sc.ResourceID,
			&sc.IntervalSeconds,
			&sc.TimeLimitSeconds,
			&sc.WaitingExpirySeconds,
			&sc.EndsAt,
			&sc.Enabled,
			&sc.NextRunAt,
			&sc.CreatedAt,
			&sc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *ScheduleStoreImpl) ListDueSchedules(ctx context.Context, now time.Time) ([]models.Schedule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id::text,
			entity_id::text,
			resource_id,
			interval_seconds,
			time_limit_seconds,
			waiting_expiry_seconds,
			ends_at,
			enabled,
			next_run_at,
			created_at,
			updated_at
		FROM schedules
		WHERE enabled = true AND next_run_at <= $1 AND (ends_at IS NULL OR ends_at > $1)
		ORDER BY next_run_at ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Schedule, 0)
	for rows.Next() {
		var sc models.Schedule
		if err := rows.Scan(
			&sc.ID,
			&sc.EntityID,
			&sc.ResourceID,
			&sc.IntervalSeconds,
			&sc.TimeLimitSeconds,
			&sc.WaitingExpirySeconds,
			&sc.EndsAt,
			&sc.Enabled,
			&sc.NextRunAt,
			&sc.CreatedAt,
			&sc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *ScheduleStoreImpl) UpdateSchedule(ctx context.Context, id string, in models.UpdateScheduleRequest) (models.Schedule, error) {
	var out models.Schedule
	err := s.db.QueryRowContext(ctx, `
		UPDATE schedules
		SET
			resource_id = COALESCE($2, resource_id),
			interval_seconds = COALESCE($3, interval_seconds),
			time_limit_seconds = COALESCE($4, time_limit_seconds),
			waiting_expiry_seconds = COALESCE($5, waiting_expiry_seconds),
			ends_at = COALESCE($6, ends_at),
			enabled = COALESCE($7, enabled),
			next_run_at = COALESCE($8, next_run_at),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING
			id::text,
			entity_id::text,
			resource_id,
			interval_seconds,
			time_limit_seconds,
			waiting_expiry_seconds,
			ends_at,
			enabled,
			next_run_at,
			created_at,
			updated_at
	`, id, in.ResourceID, in.IntervalSeconds, in.TimeLimitSeconds, in.WaitingExpirySeconds, in.EndsAt, in.Enabled, in.NextRunAt).Scan(
		&out.ID,
		&out.EntityID,
		&out.ResourceID,
		&out.IntervalSeconds,
		&out.TimeLimitSeconds,
		&out.WaitingExpirySeconds,
		&out.EndsAt,
		&out.Enabled,
		&out.NextRunAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Schedule{}, ErrNotFound
		}
		return models.Schedule{}, err
	}
	return out, nil
}

func (s *ScheduleStoreImpl) UpdateScheduleNextRunAt(ctx context.Context, id string, nextRunAt time.Time) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE schedules
		SET next_run_at = $2, updated_at = now()
		WHERE id = $1::uuid
	`, id, nextRunAt)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ScheduleStoreImpl) GetSchedulesMetrics(ctx context.Context, now time.Time) (models.SchedulesMetricsResponse, error) {
	type aggRow struct {
		ScheduleID string
		Fired      int64
		Expired    int64
		Completed  int64
		Within     int64
		AvgA2S     sql.NullFloat64
		AvgA2C     sql.NullFloat64
		AvgA2E     sql.NullFloat64
	}
	agg := make(map[string]aggRow)

	// Aggregate scheduled-node outcomes and durations using nodes + node_logs.
	rows, err := s.db.QueryContext(ctx, `
		WITH schedule_nodes AS (
			SELECT id, schedule_id, assigned_at, due_at, expired, expired_at
			FROM nodes
			WHERE schedule_id IS NOT NULL
		),
		svc_ts AS (
			SELECT node_id, MIN(ts) AS service_ts
			FROM node_logs
			WHERE action = 'moved_to_service_queue'
			GROUP BY node_id
		),
		comp_ts AS (
			SELECT node_id, MIN(ts) AS completed_ts
			FROM node_logs
			WHERE action = 'completed'
			GROUP BY node_id
		),
		node_metrics AS (
			SELECT
				sn.schedule_id::text AS schedule_id,
				sn.assigned_at,
				sn.due_at,
				sn.expired,
				sn.expired_at,
				st.service_ts,
				ct.completed_ts
			FROM schedule_nodes sn
			LEFT JOIN svc_ts st ON st.node_id = sn.id
			LEFT JOIN comp_ts ct ON ct.node_id = sn.id
		)
		SELECT
			schedule_id,
			COUNT(*) AS fired_count,
			COUNT(*) FILTER (WHERE expired) AS expired_count,
			COUNT(*) FILTER (WHERE NOT expired AND completed_ts IS NOT NULL) AS completed_count,
			COUNT(*) FILTER (WHERE NOT expired AND completed_ts IS NOT NULL AND due_at IS NOT NULL AND completed_ts <= due_at) AS within_limit_count,
			AVG(EXTRACT(EPOCH FROM (service_ts - assigned_at)) * 1000.0) FILTER (WHERE service_ts IS NOT NULL AND assigned_at IS NOT NULL) AS avg_assigned_to_allocate_ms,
			AVG(EXTRACT(EPOCH FROM (completed_ts - assigned_at)) * 1000.0) FILTER (WHERE completed_ts IS NOT NULL AND assigned_at IS NOT NULL) AS avg_assigned_to_complete_ms,
			AVG(EXTRACT(EPOCH FROM (expired_at - assigned_at)) * 1000.0) FILTER (WHERE expired AND expired_at IS NOT NULL AND assigned_at IS NOT NULL) AS avg_assigned_to_expired_ms
		FROM node_metrics
		GROUP BY schedule_id
	`)
	if err != nil {
		return models.SchedulesMetricsResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var r aggRow
		if err := rows.Scan(&r.ScheduleID, &r.Fired, &r.Expired, &r.Completed, &r.Within, &r.AvgA2S, &r.AvgA2C, &r.AvgA2E); err != nil {
			return models.SchedulesMetricsResponse{}, err
		}
		agg[r.ScheduleID] = r
	}
	if err := rows.Err(); err != nil {
		return models.SchedulesMetricsResponse{}, err
	}

	// Join with schedules table for config fields.
	sRows, err := s.db.QueryContext(ctx, `
		SELECT
			id::text,
			entity_id::text,
			resource_id,
			interval_seconds,
			time_limit_seconds,
			waiting_expiry_seconds,
			ends_at,
			enabled,
			next_run_at,
			created_at,
			updated_at
		FROM schedules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return models.SchedulesMetricsResponse{}, err
	}
	defer sRows.Close()

	out := models.SchedulesMetricsResponse{
		Totals: models.SchedulesMetricsTotals{Now: now},
	}

	var (
		sumA2S float64
		sumA2C float64
		sumA2E float64
		cntA2S int64
		cntA2C int64
		cntA2E int64
	)

	for sRows.Next() {
		var sc models.ScheduleMetrics
		if err := sRows.Scan(
			&sc.ScheduleID,
			&sc.EntityID,
			&sc.ResourceID,
			&sc.IntervalSeconds,
			&sc.TimeLimitSeconds,
			&sc.WaitingExpirySeconds,
			&sc.EndsAt,
			&sc.Enabled,
			&sc.NextRunAt,
			&sc.CreatedAt,
			&sc.UpdatedAt,
		); err != nil {
			return models.SchedulesMetricsResponse{}, err
		}

		out.Totals.TotalSchedules++
		if sc.Enabled {
			out.Totals.EnabledSchedules++
		}
		if sc.EndsAt != nil && !sc.EndsAt.After(now) {
			out.Totals.EndedSchedules++
		}

		if a, ok := agg[sc.ScheduleID]; ok {
			sc.FiredCount = a.Fired
			sc.ExpiredCount = a.Expired
			sc.CompletedCount = a.Completed
			sc.CompletedWithinTimeLimitCount = a.Within

			out.Totals.FiredCount += a.Fired
			out.Totals.ExpiredCount += a.Expired
			out.Totals.CompletedCount += a.Completed
			out.Totals.CompletedWithinTimeLimitCount += a.Within

			if a.AvgA2S.Valid && !math.IsNaN(a.AvgA2S.Float64) && !math.IsInf(a.AvgA2S.Float64, 0) {
				v := int64(a.AvgA2S.Float64 + 0.5)
				sc.AvgAssignedToAllocateMS = &v
				sumA2S += a.AvgA2S.Float64
				cntA2S++
			}
			if a.AvgA2C.Valid && !math.IsNaN(a.AvgA2C.Float64) && !math.IsInf(a.AvgA2C.Float64, 0) {
				v := int64(a.AvgA2C.Float64 + 0.5)
				sc.AvgAssignedToCompleteMS = &v
				sumA2C += a.AvgA2C.Float64
				cntA2C++
			}
			if a.AvgA2E.Valid && !math.IsNaN(a.AvgA2E.Float64) && !math.IsInf(a.AvgA2E.Float64, 0) {
				v := int64(a.AvgA2E.Float64 + 0.5)
				sc.AvgAssignedToExpiredMS = &v
				sumA2E += a.AvgA2E.Float64
				cntA2E++
			}
		}

		out.Schedules = append(out.Schedules, sc)
	}
	if err := sRows.Err(); err != nil {
		return models.SchedulesMetricsResponse{}, err
	}

	if cntA2S > 0 {
		v := int64(sumA2S/float64(cntA2S) + 0.5)
		out.Totals.AvgAssignedToAllocateMS = &v
	}
	if cntA2C > 0 {
		v := int64(sumA2C/float64(cntA2C) + 0.5)
		out.Totals.AvgAssignedToCompleteMS = &v
	}
	if cntA2E > 0 {
		v := int64(sumA2E/float64(cntA2E) + 0.5)
		out.Totals.AvgAssignedToExpiredMS = &v
	}

	return out, nil
}

