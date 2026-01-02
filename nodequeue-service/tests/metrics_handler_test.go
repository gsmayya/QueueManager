package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	queueservicepkg "nodequeue-service/queueservice"
	resourcepkg "nodequeue-service/resource"
)

func TestNodesMetricsHandler_CompletesAndComputesWaitingSegments(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := resourcepkg.NewResource("resource-1", 1)
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

	var resp queueservicepkg.NodesMetricsResponse
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
	r1 := resourcepkg.NewResource("resource-1", 1)
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

	var resp queueservicepkg.NodesMetricsResponse
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
