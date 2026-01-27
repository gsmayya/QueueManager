package tests

import (
	"context"
	"queue-common/models"
	"testing"
	"time"

	storepkg "queue-common/store"
	queueservicepkg "queue-service/queueservice"
)

type stubStore struct {
	nodes  []storepkg.PersistedNode
	states map[string]storepkg.NodeState
}

func (s *stubStore) ListResources(ctx context.Context) ([]*models.Resource, error) {
	return nil, nil
}

func (s *stubStore) ListNodes(ctx context.Context) ([]storepkg.PersistedNode, error) {
	return s.nodes, nil
}

func (s *stubStore) ListLatestNodeStates(ctx context.Context) (map[string]storepkg.NodeState, error) {
	return s.states, nil
}

func (s *stubStore) ListNodeLogs(ctx context.Context, nodeIDs []string) (map[string][]storepkg.NodeLogRow, error) {
	return map[string][]storepkg.NodeLogRow{}, nil
}

func (s *stubStore) EnsureEntity(ctx context.Context, entityID, entityName string, createdAt time.Time) error {
	return nil
}

func (s *stubStore) PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName, nodeName string, createdAt time.Time) error {
	return nil
}
func (s *stubStore) UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error {
	return nil
}
func (s *stubStore) MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error {
	return nil
}
func (s *stubStore) InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error {
	return nil
}

func (s *stubStore) UpdateNodeScheduling(ctx context.Context, nodeID string, scheduleID *string, timeLimitSeconds *int, assignedAt, dueAt *time.Time, delayFlag *bool) error {
	return nil
}
func (s *stubStore) UpdateNodeExpiry(ctx context.Context, nodeID string, waitingExpirySeconds *int, expiresAt *time.Time) error {
	return nil
}
func (s *stubStore) MarkNodeExpired(ctx context.Context, nodeID string, expiredAt time.Time) error {
	return nil
}
func (s *stubStore) HasActiveNodeForSchedule(ctx context.Context, scheduleID string) (bool, error) {
	return false, nil
}
func (s *stubStore) MarkOverdueNodes(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

func TestRestoreFromStore_RebuildsQueuesAndOrder(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	store := &stubStore{
		nodes: []storepkg.PersistedNode{
			{NodeID: "n_wait_1", EntityID: "e1-id", EntityName: "e1", NodeName: "0001", ResourceID: ptr("Room 1"), Completed: false, CreatedAt: base.Add(1 * time.Minute)},
			{NodeID: "n_wait_2", EntityID: "e2-id", EntityName: "e2", NodeName: "0002", ResourceID: ptr("Room 1"), Completed: false, CreatedAt: base.Add(2 * time.Minute)},
			{NodeID: "n_svc", EntityID: "e3-id", EntityName: "e3", NodeName: "0003", ResourceID: ptr("Room 1"), Completed: false, CreatedAt: base.Add(3 * time.Minute)},
			{NodeID: "n_room2", EntityID: "e4-id", EntityName: "e4", NodeName: "0004", ResourceID: ptr("Room 2"), Completed: false, CreatedAt: base.Add(4 * time.Minute)},
			{NodeID: "n_unassigned", EntityID: "e5-id", EntityName: "e5", NodeName: "0005", ResourceID: nil, Completed: false, CreatedAt: base.Add(5 * time.Minute)},
		},
		states: map[string]storepkg.NodeState{
			// Waiting order should be by TS asc: n_wait_2 (ts=10s) then n_wait_1 (ts=20s)
			"n_wait_1": {Queue: storepkg.QueueKindWaiting, TS: base.Add(20 * time.Second)},
			"n_wait_2": {Queue: storepkg.QueueKindWaiting, TS: base.Add(10 * time.Second)},
			"n_svc":    {Queue: storepkg.QueueKindService, TS: base.Add(30 * time.Second)},
			// No explicit state for n_room2 => defaults to waiting with CreatedAt ordering.
		},
	}

	qs := queueservicepkg.NewQueueServiceWithStore(store, nil)
	qs.AddResource(models.NewResource("Room 1", 5))
	qs.AddResource(models.NewResource("Room 2", 5))

	if err := qs.RestoreFromStore(context.Background()); err != nil {
		t.Fatalf("RestoreFromStore failed: %v", err)
	}

	nodes := qs.ListNodes()
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes restored, got %d", len(nodes))
	}

	room1, err := qs.GetResource("Room 1")
	if err != nil {
		t.Fatalf("expected Room 1 resource, got err: %v", err)
	}
	if len(room1.Nodes) != 1 || room1.Nodes[0].ID != "n_svc" {
		t.Fatalf("expected service queue [n_svc], got %v", ids(room1.Nodes))
	}
	if len(room1.WaitingQueue) != 2 || room1.WaitingQueue[0].ID != "n_wait_2" || room1.WaitingQueue[1].ID != "n_wait_1" {
		t.Fatalf("expected waiting queue [n_wait_2 n_wait_1], got %v", ids(room1.WaitingQueue))
	}

	room2, err := qs.GetResource("Room 2")
	if err != nil {
		t.Fatalf("expected Room 2 resource, got err: %v", err)
	}
	if len(room2.WaitingQueue) != 1 || room2.WaitingQueue[0].ID != "n_room2" {
		t.Fatalf("expected Room 2 waiting queue [n_room2], got %v", ids(room2.WaitingQueue))
	}
}
