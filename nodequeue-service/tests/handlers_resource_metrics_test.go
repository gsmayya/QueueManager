package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	queueservicepkg "nodequeue-service/queueservice"
	resourcepkg "nodequeue-service/resource"
)

func TestResourcesMetricsHandler_MethodNotAllowed(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	req := httptest.NewRequest(http.MethodPost, "/resources/metrics", nil)
	w := httptest.NewRecorder()
	qs.ResourcesMetricsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestResourcesMetricsHandler_ReportsAllResources_AndCounts(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := resourcepkg.NewResource("resource-1", 1)
	r2 := resourcepkg.NewResource("resource-2", 1)
	qs.AddResource(r1)
	qs.AddResource(r2)

	// r1 activity
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

	req := httptest.NewRequest(http.MethodGet, "/resources/metrics", nil)
	w := httptest.NewRecorder()
	qs.ResourcesMetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp queueservicepkg.ResourcesSessionMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Resources) != 2 {
		t.Fatalf("expected 2 resources in response, got %d", len(resp.Resources))
	}

	byID := map[string]queueservicepkg.ResourceSessionMetrics{}
	for _, m := range resp.Resources {
		byID[m.ResourceID] = m
	}

	m1, ok := byID[r1.ID]
	if !ok {
		t.Fatalf("expected resource %s metrics", r1.ID)
	}
	if m1.TotalAdded != 1 {
		t.Fatalf("expected r1 total_added=1, got %d", m1.TotalAdded)
	}
	if m1.TotalAllocated != 1 {
		t.Fatalf("expected r1 total_allocated=1, got %d", m1.TotalAllocated)
	}
	if m1.CurrentAllocated != 1 || m1.CurrentWaiting != 0 {
		t.Fatalf("expected r1 current_waiting=0 current_allocated=1, got %d/%d", m1.CurrentWaiting, m1.CurrentAllocated)
	}
	if m1.WaitingSegmentsCount != 1 {
		t.Fatalf("expected r1 waiting_segments_count=1, got %d", m1.WaitingSegmentsCount)
	}
	if m1.ServiceSegmentsCount != 1 {
		t.Fatalf("expected r1 service_segments_count=1, got %d", m1.ServiceSegmentsCount)
	}
	if m1.AvgWaitingTimeMS < 0 || m1.AvgServiceTimeMS < 0 {
		t.Fatalf("expected non-negative averages, got waiting=%d service=%d", m1.AvgWaitingTimeMS, m1.AvgServiceTimeMS)
	}

	m2, ok := byID[r2.ID]
	if !ok {
		t.Fatalf("expected resource %s metrics", r2.ID)
	}
	if m2.TotalAdded != 0 || m2.TotalAllocated != 0 {
		t.Fatalf("expected r2 totals=0, got added=%d alloc=%d", m2.TotalAdded, m2.TotalAllocated)
	}
	if m2.CurrentAllocated != 0 || m2.CurrentWaiting != 0 {
		t.Fatalf("expected r2 current=0/0, got %d/%d", m2.CurrentWaiting, m2.CurrentAllocated)
	}
}

func TestResourcesMetricsHandler_CountsRevisits(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	r1 := resourcepkg.NewResource("resource-1", 5)
	qs.AddResource(r1)

	n, _ := qs.CreateNode("entity-1")
	_ = qs.MoveNode(n.ID, r1.ID)
	_ = qs.MoveNode(n.ID, r1.ID) // revisit should count as another add

	req := httptest.NewRequest(http.MethodGet, "/resources/metrics", nil)
	w := httptest.NewRecorder()
	qs.ResourcesMetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp queueservicepkg.ResourcesSessionMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Resources) != 1 {
		t.Fatalf("expected 1 resource in response, got %d", len(resp.Resources))
	}
	m := resp.Resources[0]
	if m.ResourceID != r1.ID {
		t.Fatalf("expected resource_id %s, got %s", r1.ID, m.ResourceID)
	}
	if m.TotalAdded != 2 {
		t.Fatalf("expected total_added=2, got %d", m.TotalAdded)
	}
	if m.WaitingSegmentsCount != 2 {
		t.Fatalf("expected waiting_segments_count=2, got %d", m.WaitingSegmentsCount)
	}
}
