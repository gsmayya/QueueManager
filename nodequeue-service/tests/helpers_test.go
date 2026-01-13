package tests

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	nodepkg "nodequeue-service/node"
	queueservicepkg "nodequeue-service/queueservice"
	resourcepkg "nodequeue-service/resource"
	storepkg "nodequeue-service/store"
)

func mustJSONDecode[T any](t *testing.T, rr *httptest.ResponseRecorder, out *T) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func mustCreateQSWithResources(t *testing.T, resources ...*resourcepkg.Resource) *queueservicepkg.QueueService {
	t.Helper()
	qs := queueservicepkg.NewQueueService()
	for _, r := range resources {
		qs.AddResource(r)
	}
	return qs
}

func ptr[T any](v T) *T { return &v } // used by restore tests

func ids(ns []*nodepkg.Node) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		out = append(out, n.ID)
	}
	return out
}

// stubReadLogsStore is used to test DB-backed metrics calculations without a real DB.
// Other methods are no-ops/empty to satisfy storepkg.Store.
type stubReadLogsStore struct {
	logsByNode map[string][]storepkg.NodeLogRow
}

func (s *stubReadLogsStore) ListResources(ctx context.Context) ([]*resourcepkg.Resource, error) {
	return nil, nil
}
func (s *stubReadLogsStore) ListNodes(ctx context.Context) ([]storepkg.PersistedNode, error) {
	return nil, nil
}
func (s *stubReadLogsStore) ListLatestNodeStates(ctx context.Context) (map[string]storepkg.NodeState, error) {
	return map[string]storepkg.NodeState{}, nil
}
func (s *stubReadLogsStore) ListNodeLogs(ctx context.Context, nodeIDs []string) (map[string][]storepkg.NodeLogRow, error) {
	out := make(map[string][]storepkg.NodeLogRow, len(nodeIDs))
	for _, id := range nodeIDs {
		out[id] = s.logsByNode[id]
	}
	return out, nil
}
func (s *stubReadLogsStore) PersistNodeCreated(ctx context.Context, nodeID, entityID, entityName, nodeName string, createdAt time.Time) error {
	return nil
}
func (s *stubReadLogsStore) UpdateNodeResource(ctx context.Context, nodeID string, resourceID *string) error {
	return nil
}
func (s *stubReadLogsStore) MarkNodeCompleted(ctx context.Context, nodeID string, completed bool) error {
	return nil
}
func (s *stubReadLogsStore) InsertNodeLog(ctx context.Context, nodeID, action string, resourceID *string, ts time.Time) error {
	return nil
}

var _ storepkg.Store = (*stubReadLogsStore)(nil)
