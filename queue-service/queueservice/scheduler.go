package queueservice

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"queue-common/logging"
	"queue-common/models"
)

var schedRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func (qs *QueueService) RunDueSchedules(ctx context.Context, now time.Time) {
	if qs.schedstore == nil || qs.nodestore == nil {
		return
	}
	schedules, err := qs.schedstore.ListDueSchedules(ctx, now)
	if err != nil {
		logging.Debugf("[DB] ListDueSchedules failed: %v", err)
		return
	}
	for _, sc := range schedules {
		qs.runOneSchedule(ctx, now, sc)

		// Advance next_run_at until it's in the future (prevents rapid backlog creation after downtime).
		next := sc.NextRunAt
		step := time.Duration(sc.IntervalSeconds) * time.Second
		if step <= 0 {
			step = time.Second
		}
		for !next.After(now) {
			next = next.Add(step)
		}
		qs.bestEffortPersist(ctx, "UpdateScheduleNextRunAt", func(ctx context.Context) error {
			return qs.schedstore.UpdateScheduleNextRunAt(ctx, sc.ID, next)
		})
	}
}

func (qs *QueueService) runOneSchedule(ctx context.Context, now time.Time, sc models.Schedule) {
	overflow := false
	if exists, err := qs.nodestore.HasActiveNodeForSchedule(ctx, sc.ID); err == nil && exists {
		overflow = true
	}

	ent, err := fetchMasterEntityByID(ctx, sc.EntityID)
	if err != nil {
		logging.Errorf("schedule_run entity_fetch_failed schedule_id=%s entity_id=%s err=%v", sc.ID, sc.EntityID, err)
		return
	}

	nodeName := qs.generateUniqueNodeName()
	n, err := qs.CreateNodeForEntity(sc.EntityID, ent.Name, nodeName)
	if err != nil {
		logging.Errorf("schedule_run node_create_failed schedule_id=%s err=%v", sc.ID, err)
		return
	}

	assignedAt := now
	if err := qs.MoveNode(n.ID, sc.ResourceID); err != nil {
		logging.Errorf("schedule_run node_move_failed schedule_id=%s node_id=%s resource_id=%s err=%v", sc.ID, n.ID, sc.ResourceID, err)
		return
	}

	dueAt := assignedAt.Add(time.Duration(sc.TimeLimitSeconds) * time.Second)
	expiresAt := assignedAt.Add(time.Duration(sc.WaitingExpirySeconds) * time.Second)
	// Do NOT pre-mark newly created scheduled nodes as delayed based on prior instances.
	// Delay is set only when due_at is exceeded (DelayFlagScan).
	delay := false

	// Update in-memory fields.
	qs.mu.Lock()
	if nn, ok := qs.nodes[n.ID]; ok {
		sid := sc.ID
		tls := sc.TimeLimitSeconds
		wes := sc.WaitingExpirySeconds
		nn.ScheduleID = &sid
		nn.TimeLimitSeconds = &tls
		nn.WaitingExpirySeconds = &wes
		nn.AssignedAt = &assignedAt
		nn.DueAt = &dueAt
		nn.ExpiresAt = &expiresAt
		nn.DelayFlag = delay
	}
	qs.mu.Unlock()

	// Persist schedule metadata to node row (best-effort).
	sid := sc.ID
	tls := sc.TimeLimitSeconds
	df := delay
	qs.bestEffortPersist(ctx, "UpdateNodeScheduling", func(ctx context.Context) error {
		return qs.nodestore.UpdateNodeScheduling(ctx, n.ID, &sid, &tls, &assignedAt, &dueAt, &df)
	})
	wes := sc.WaitingExpirySeconds
	qs.bestEffortPersist(ctx, "UpdateNodeExpiry", func(ctx context.Context) error {
		return qs.nodestore.UpdateNodeExpiry(ctx, n.ID, &wes, &expiresAt)
	})

	// Log schedule addition and overflow marker for metrics.
	rid := sc.ResourceID
	qs.bestEffortPersist(ctx, "InsertNodeLog(scheduled)", func(ctx context.Context) error {
		return qs.nodestore.InsertNodeLog(ctx, n.ID, "scheduled", &rid, assignedAt)
	})
	if overflow {
		qs.bestEffortPersist(ctx, "InsertNodeLog(schedule_overflow)", func(ctx context.Context) error {
			return qs.nodestore.InsertNodeLog(ctx, n.ID, "schedule_overflow", &rid, assignedAt)
		})
	}

	logging.Infof(
		"schedule_fired schedule_id=%s node_id=%s resource_id=%s due_at=%s expires_at=%s overflow=%t",
		sc.ID, n.ID, sc.ResourceID, dueAt.UTC().Format(time.RFC3339), expiresAt.UTC().Format(time.RFC3339), overflow,
	)
}

// ExpiryScan auto-expires scheduled nodes that are still in WAITING past expiry conditions.
// Expiry applies only to waiting nodes (not nodes already allocated into service).
func (qs *QueueService) ExpiryScan(ctx context.Context, now time.Time) {
	if qs.schedstore == nil || qs.nodestore == nil {
		return
	}

	// Snapshot schedules for ends_at and waiting_expiry_seconds.
	schedules, err := qs.schedstore.ListSchedules(ctx)
	if err != nil {
		logging.Debugf("[DB] ListSchedules failed (expiry scan skipped): %v", err)
		return
	}
	type schedInfo struct {
		waitingExpirySeconds int
		endsAt              *time.Time
		enabled             bool
	}
	byID := make(map[string]schedInfo, len(schedules))
	for _, s := range schedules {
		byID[s.ID] = schedInfo{
			waitingExpirySeconds: s.WaitingExpirySeconds,
			endsAt:              s.EndsAt,
			enabled:             s.Enabled,
		}
		// Best-effort: auto-disable schedules that are past ends_at.
		if s.Enabled && s.EndsAt != nil && !s.EndsAt.After(now) {
			id := s.ID
			enabled := false
			qs.bestEffortPersist(ctx, "DisableEndedSchedule", func(ctx context.Context) error {
				_, err := qs.schedstore.UpdateSchedule(ctx, id, models.UpdateScheduleRequest{Enabled: &enabled})
				return err
			})
		}
	}

	qs.mu.Lock()
	defer qs.mu.Unlock()

	for _, r := range qs.resources {
		waiting := r.WaitingQueue
		if len(waiting) == 0 {
			continue
		}
		// Iterate backwards so we can remove safely.
		for i := len(waiting) - 1; i >= 0; i-- {
			n := waiting[i]
			if n == nil || n.Completed || n.Expired || n.ScheduleID == nil {
				continue
			}
			sid := *n.ScheduleID
			info, ok := byID[sid]
			if !ok {
				continue
			}

			expire := false

			// Ends-at expiry: expire waiting nodes when schedule has ended.
			if info.endsAt != nil && !info.endsAt.After(now) {
				expire = true
			}

			// Per-node expiry: expires_at if present, else compute from assigned_at + waiting_expiry_seconds.
			var ex *time.Time = n.ExpiresAt
			if ex == nil && n.AssignedAt != nil {
				wes := info.waitingExpirySeconds
				if n.WaitingExpirySeconds != nil {
					wes = *n.WaitingExpirySeconds
				}
				t := n.AssignedAt.Add(time.Duration(wes) * time.Second)
				ex = &t
				n.ExpiresAt = ex
				if n.WaitingExpirySeconds == nil {
					n.WaitingExpirySeconds = &wes
				}
			}

			// If we still can't determine expiry time and the schedule isn't ended, skip.
			// This avoids accidentally expiring a newer node that hasn't had assigned/expires set yet.
			if ex == nil && (info.endsAt == nil || info.endsAt.After(now)) {
				continue
			}
			if ex != nil && !ex.After(now) {
				expire = true
			}

			if !expire {
				continue
			}

			prevRID := n.ResourceID

			// In-memory: remove from queues and mark expired/completed.
			r.RemoveNode(n.ID)
			n.Completed = true
			n.Expired = true
			n.DelayFlag = n.DelayFlag || (n.DueAt != nil && n.DueAt.Before(now))
			n.ResourceID = ""
			n.AddLog("schedule_expired", prevRID)
			ts := now
			n.ExpiredAt = &ts

			// Persist: mark expired and log.
			nodeID := n.ID
			rid := prevRID
			qs.bestEffortPersist(ctx, "MarkNodeExpired", func(ctx context.Context) error {
				return qs.nodestore.MarkNodeExpired(ctx, nodeID, now)
			})
			qs.bestEffortPersist(ctx, "InsertNodeLog(schedule_expired)", func(ctx context.Context) error {
				return qs.nodestore.InsertNodeLog(ctx, nodeID, "schedule_expired", &rid, now)
			})

			logging.Infof("schedule_expired node_id=%s schedule_id=%s prev_resource=%q", nodeID, sid, prevRID)
		}
	}
}

func (qs *QueueService) DelayFlagScan(ctx context.Context, now time.Time) {
	// Persist first (best-effort), then reconcile in-memory state.
	if qs.nodestore != nil {
		qs.bestEffortPersist(ctx, "MarkOverdueNodes", func(ctx context.Context) error {
			_, err := qs.nodestore.MarkOverdueNodes(ctx, now)
			return err
		})
	}

	qs.mu.Lock()
	for _, n := range qs.nodes {
		if n.Completed || n.DelayFlag {
			continue
		}
		if n.DueAt != nil && n.DueAt.Before(now) {
			n.DelayFlag = true
		}
	}
	qs.mu.Unlock()
}

func (qs *QueueService) generateUniqueNodeName() string {
	// 10k space; retry a bit.
	for i := 0; i < 20000; i++ {
		cand := fmt.Sprintf("%04d", schedRand.Intn(10000))
		if qs.nodeNameAvailable(cand) {
			return cand
		}
	}
	// Fallback: collisions are extremely unlikely; last resort is still a 4-digit string.
	return fmt.Sprintf("%04d", schedRand.Intn(10000))
}

func (qs *QueueService) nodeNameAvailable(name string) bool {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	for _, n := range qs.nodes {
		if n.NodeName == name {
			return false
		}
	}
	return true
}

