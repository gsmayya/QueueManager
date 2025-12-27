package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateNodeHandler(t *testing.T) {
	qs := NewQueueService()

	// Test successful creation
	reqBody := CreateNodeRequest{EntityName: "test-entity"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/nodes", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	qs.CreateNodeHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var node Node
	if err := json.NewDecoder(w.Body).Decode(&node); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if node.Entity == nil || node.Entity.Name != "test-entity" {
		t.Errorf("Expected entity name 'test-entity', got '%s'", node.Entity.Name)
	}
	if len(node.Log) == 0 || node.Log[0].Action != "created" {
		t.Error("Expected node to include a creation log entry")
	}

	// Test missing entity_name
	reqBody = CreateNodeRequest{EntityName: ""}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.CreateNodeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test invalid method
	req = httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w = httptest.NewRecorder()

	qs.CreateNodeHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestMoveNodeHandler(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 3)
	resource2 := NewResource("resource-2", 2)
	qs.AddResource(resource1)
	qs.AddResource(resource2)

	node, _ := qs.CreateNode("test-entity")

	// Test successful move
	reqBody := MoveNodeRequest{TargetResourceID: "resource-1"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/nodes/"+node.ID+"/move", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, node.ID)

	t.Logf("HTTP Status: %d, Body: %s", w.Code, w.Body.String())
	var movedNode Node
	if err := json.NewDecoder(w.Body).Decode(&movedNode); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if movedNode.ResourceID != "resource-1" {
		t.Errorf("Expected ResourceID 'resource-1', got '%s'", movedNode.ResourceID)
	}

	// Test missing target_resource_id
	reqBody = MoveNodeRequest{TargetResourceID: ""}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/"+node.ID+"/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, node.ID)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test non-existent node
	reqBody = MoveNodeRequest{TargetResourceID: "resource-1"}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/non-existent/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, "non-existent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	// Test non-existent resource
	reqBody = MoveNodeRequest{TargetResourceID: "non-existent"}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/"+node.ID+"/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, node.ID)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCompleteNodeHandler(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 3)
	qs.AddResource(resource1)

	node, _ := qs.CreateNode("test-entity")
	qs.MoveNode(node.ID, "resource-1")

	// Test successful completion
	req := httptest.NewRequest(http.MethodPost, "/nodes/"+node.ID+"/complete", nil)
	w := httptest.NewRecorder()

	qs.CompleteNodeHandler(w, req, node.ID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var completedNode Node
	if err := json.NewDecoder(w.Body).Decode(&completedNode); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !completedNode.Completed {
		t.Error("Node should be marked as completed")
	}

	// Test non-existent node
	req = httptest.NewRequest(http.MethodPost, "/nodes/non-existent/complete", nil)
	w = httptest.NewRecorder()

	qs.CompleteNodeHandler(w, req, "non-existent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestAllocateNodeHandler(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 1)
	qs.AddResource(resource1)

	node1, _ := qs.CreateNode("entity-1")
	node2, _ := qs.CreateNode("entity-2")
	qs.MoveNode(node1.ID, "resource-1")
	qs.MoveNode(node2.ID, "resource-1")

	// Allocate first node - should succeed
	req := httptest.NewRequest(http.MethodPost, "/nodes/"+node1.ID+"/allocate", nil)
	w := httptest.NewRecorder()
	qs.AllocateNodeHandler(w, req, node1.ID)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Allocate second node - should fail (capacity exceeded)
	req = httptest.NewRequest(http.MethodPost, "/nodes/"+node2.ID+"/allocate", nil)
	w = httptest.NewRecorder()
	qs.AllocateNodeHandler(w, req, node2.ID)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetNodeHandler(t *testing.T) {
	qs := NewQueueService()
	node, _ := qs.CreateNode("test-entity")

	// Test successful retrieval
	req := httptest.NewRequest(http.MethodGet, "/nodes/"+node.ID, nil)
	w := httptest.NewRecorder()

	qs.GetNodeHandler(w, req, node.ID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var retrievedNode Node
	if err := json.NewDecoder(w.Body).Decode(&retrievedNode); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if retrievedNode.ID != node.ID {
		t.Errorf("Node ID mismatch")
	}

	// Test non-existent node
	req = httptest.NewRequest(http.MethodGet, "/nodes/non-existent", nil)
	w = httptest.NewRecorder()

	qs.GetNodeHandler(w, req, "non-existent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestListNodesHandler(t *testing.T) {
	qs := NewQueueService()
	qs.CreateNode("entity-1")
	qs.CreateNode("entity-2")

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w := httptest.NewRecorder()

	qs.ListNodesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var nodes []Node
	if err := json.NewDecoder(w.Body).Decode(&nodes); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
}

func TestListResourcesHandler(t *testing.T) {
	qs := NewQueueService()
	qs.AddResource(NewResource("resource-1", 5))
	qs.AddResource(NewResource("resource-2", 3))

	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	w := httptest.NewRecorder()

	qs.ListResourcesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resources []Resource
	if err := json.NewDecoder(w.Body).Decode(&resources); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}
