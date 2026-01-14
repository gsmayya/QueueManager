package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"queue-common/models"
	cmstore "queue-common/store"
	"testing"
	"time"

	queueservicepkg "queue-service/queueservice"
)

func TestNodesMetricsHandler_CompletesAndComputesWaitingSegments(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := models.NewResource("resource-1", 1)
	qs.AddResource(r1)

	n, err := qs.CreateNode("entity-1")
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}
	if err := qs.MoveNode(n.ID, r1.ID); err != nil {
		t.Fatalf("MoveNode failed: %v", err)
	}
	if err := qs.AllocateNode(n.ID); err != nil {
		t.Fatalf("AllocateNode failed: %v", err)
	}
	if err := qs.CompleteNode(n.ID); err != nil {
		t.Fatalf("CompleteNode failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nodes/metrics", nil)
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.NodesMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.CompletedNodes) != 1 {
		t.Fatalf("expected 1 completed node, got %d", len(resp.CompletedNodes))
	}
	if len(resp.ActiveNodes) != 0 {
		t.Fatalf("expected 0 active nodes, got %d", len(resp.ActiveNodes))
	}

	m := resp.CompletedNodes[0]
	if m.ID != n.ID {
		t.Fatalf("expected completed node id %s, got %s", n.ID, m.ID)
	}
	if m.TotalTimeInSystemMS < 0 {
		t.Fatalf("expected non-negative total_time_in_system_ms, got %d", m.TotalTimeInSystemMS)
	}
	if len(m.WaitingSegments) != 1 {
		t.Fatalf("expected 1 waiting segment, got %d", len(m.WaitingSegments))
	}
	seg := m.WaitingSegments[0]
	if seg.ResourceID != r1.ID {
		t.Fatalf("expected segment resource_id %s, got %s", r1.ID, seg.ResourceID)
	}
	if seg.DurationMS < 0 {
		t.Fatalf("expected non-negative duration_ms, got %d", seg.DurationMS)
	}
}

func TestNodesMetricsHandler_ActiveNodeHasOpenWaitingSegmentClosedAtNow(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := models.NewResource("resource-1", 1)
	qs.AddResource(r1)

	n, err := qs.CreateNode("entity-1")
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}
	if err := qs.MoveNode(n.ID, r1.ID); err != nil {
		t.Fatalf("MoveNode failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nodes/metrics", nil)
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.NodesMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.ActiveNodes) != 1 {
		t.Fatalf("expected 1 active node, got %d", len(resp.ActiveNodes))
	}
	m := resp.ActiveNodes[0]
	if len(m.WaitingSegments) != 1 {
		t.Fatalf("expected 1 waiting segment, got %d", len(m.WaitingSegments))
	}
	seg := m.WaitingSegments[0]
	if seg.EndTS.Before(seg.StartTS) {
		t.Fatalf("expected end_ts >= start_ts, got start=%v end=%v", seg.StartTS, seg.EndTS)
	}
}

func TestNodesMetricsHandler_MethodNotAllowed(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	req := httptest.NewRequest(http.MethodPost, "/nodes/metrics", nil)
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestNodesMetricsHandler_MoveWhileWaiting_ClosesAndOpensSegments(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := models.NewResource("resource-1", 5)
	r2 := models.NewResource("resource-2", 5)
	qs.AddResource(r1)
	qs.AddResource(r2)

	n, _ := qs.CreateNode("entity-1")
	if err := qs.MoveNode(n.ID, r1.ID); err != nil {
		t.Fatalf("MoveNode r1 failed: %v", err)
	}
	if err := qs.MoveNode(n.ID, r2.ID); err != nil {
		t.Fatalf("MoveNode r2 failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nodes/metrics", nil)
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.NodesMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.ActiveNodes) != 1 {
		t.Fatalf("expected 1 active node, got %d", len(resp.ActiveNodes))
	}
	m := resp.ActiveNodes[0]
	if len(m.WaitingSegments) != 2 {
		t.Fatalf("expected 2 waiting segments, got %d", len(m.WaitingSegments))
	}
	if m.WaitingSegments[0].ResourceID != r1.ID || m.WaitingSegments[1].ResourceID != r2.ID {
		t.Fatalf("expected segment resource_ids [%s %s], got [%s %s]",
			r1.ID, r2.ID, m.WaitingSegments[0].ResourceID, m.WaitingSegments[1].ResourceID)
	}
	// When moved while waiting, previous segment ends exactly when next begins.
	if !m.WaitingSegments[0].EndTS.Equal(m.WaitingSegments[1].StartTS) {
		t.Fatalf("expected segment0 end_ts == segment1 start_ts, got %v vs %v",
			m.WaitingSegments[0].EndTS, m.WaitingSegments[1].StartTS)
	}
}

func TestNodesMetricsHandler_RevisitResource_AddsNewEntries(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := models.NewResource("resource-1", 5)
	r2 := models.NewResource("resource-2", 5)
	qs.AddResource(r1)
	qs.AddResource(r2)

	n, _ := qs.CreateNode("entity-1")
	_ = qs.MoveNode(n.ID, r1.ID)
	_ = qs.AllocateNode(n.ID)
	_ = qs.MoveNode(n.ID, r2.ID)
	_ = qs.AllocateNode(n.ID)
	_ = qs.MoveNode(n.ID, r1.ID) // revisit resource-1 (waiting again)
	_ = qs.CompleteNode(n.ID)

	req := httptest.NewRequest(http.MethodGet, "/nodes/metrics", nil)
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.NodesMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.CompletedNodes) != 1 {
		t.Fatalf("expected 1 completed node, got %d", len(resp.CompletedNodes))
	}
	m := resp.CompletedNodes[0]
	if len(m.WaitingSegments) != 3 {
		t.Fatalf("expected 3 waiting segments, got %d", len(m.WaitingSegments))
	}
	got := []string{m.WaitingSegments[0].ResourceID, m.WaitingSegments[1].ResourceID, m.WaitingSegments[2].ResourceID}
	want := []string{r1.ID, r2.ID, r1.ID}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected segment resource_ids %v, got %v", want, got)
		}
	}
}

func TestNodesMetricsHandler_PrefersDBLogs_WhenAvailable(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rid := "resource-1"

	store := &stubReadLogsStore{
		logsByNode: map[string][]cmstore.NodeLogRow{},
	}

	qs := queueservicepkg.NewQueueServiceWithStore(store)
	qs.AddResource(models.NewResource(rid, 5))

	created, _ := qs.CreateNode("entity-1")
	nid := created.ID

	// Make snapshot deterministic and ensure handler classifies it as completed.
	n, _ := qs.GetNode(nid)
	n.CreatedAt = base
	n.Completed = true
	n.Log = nil

	// Provide DB logs for this node ID; handler should prefer these over in-memory logs.
	store.logsByNode[nid] = []cmstore.NodeLogRow{
		{NodeID: nid, Action: "moved_to_waiting_queue", ResourceID: &rid, TS: base.Add(1 * time.Second)},
		{NodeID: nid, Action: "moved_to_service_queue", ResourceID: &rid, TS: base.Add(3 * time.Second)},
		{NodeID: nid, Action: "completed", ResourceID: &rid, TS: base.Add(10 * time.Second)},
	}

	req := httptest.NewRequest(http.MethodGet, "/nodes/metrics", nil)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()
	qs.NodesMetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.NodesMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.CompletedNodes) != 1 {
		t.Fatalf("expected 1 completed node, got %d", len(resp.CompletedNodes))
	}
	m := resp.CompletedNodes[0]
	if len(m.WaitingSegments) != 1 {
		t.Fatalf("expected 1 waiting segment, got %d", len(m.WaitingSegments))
	}
	if m.TotalTimeInSystemMS != (10 * time.Second).Milliseconds() {
		t.Fatalf("expected total_time_in_system_ms=10000, got %d", m.TotalTimeInSystemMS)
	}
}
