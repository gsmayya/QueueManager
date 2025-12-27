package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nodequeue-service/node"
	queueservicepkg "nodequeue-service/queueservice"
	resourcepkg "nodequeue-service/resource"
)

func TestCreateNodeHandler(t *testing.T) {
	qs := queueservicepkg.NewQueueService()

	// Test successful creation
	reqBody := node.CreateNodeRequest{EntityName: "test-entity"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/nodes", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	qs.CreateNodeHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var created node.Node
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if created.Entity == nil || created.Entity.Name != "test-entity" {
		got := ""
		if created.Entity != nil {
			got = created.Entity.Name
		}
		t.Errorf("Expected entity name 'test-entity', got '%s'", got)
	}
	if len(created.Log) == 0 || created.Log[0].Action != "created" {
		t.Error("Expected node to include a creation log entry")
	}

	// Test missing entity_name
	reqBody = node.CreateNodeRequest{EntityName: ""}
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
	qs := queueservicepkg.NewQueueService()
	resource1 := resourcepkg.NewResource("resource-1", 3)
	resource2 := resourcepkg.NewResource("resource-2", 2)
	qs.AddResource(resource1)
	qs.AddResource(resource2)

	created, _ := qs.CreateNode("test-entity")

	// Test successful move
	reqBody := node.MoveNodeRequest{TargetResourceID: "resource-1"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/nodes/"+created.ID+"/move", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, created.ID)

	var moved node.Node
	if err := json.NewDecoder(w.Body).Decode(&moved); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if moved.ResourceID != "resource-1" {
		t.Errorf("Expected ResourceID 'resource-1', got '%s'", moved.ResourceID)
	}

	// Test missing target_resource_id
	reqBody = node.MoveNodeRequest{TargetResourceID: ""}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/"+created.ID+"/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, created.ID)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test non-existent node
	reqBody = node.MoveNodeRequest{TargetResourceID: "resource-1"}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/non-existent/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, "non-existent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	// Test non-existent resource
	reqBody = node.MoveNodeRequest{TargetResourceID: "non-existent"}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/nodes/"+created.ID+"/move", bytes.NewBuffer(jsonBody))
	w = httptest.NewRecorder()

	qs.MoveNodeHandler(w, req, created.ID)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCompleteNodeHandler(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	resource1 := resourcepkg.NewResource("resource-1", 3)
	qs.AddResource(resource1)

	created, _ := qs.CreateNode("test-entity")
	qs.MoveNode(created.ID, "resource-1")

	// Test successful completion
	req := httptest.NewRequest(http.MethodPost, "/nodes/"+created.ID+"/complete", nil)
	w := httptest.NewRecorder()

	qs.CompleteNodeHandler(w, req, created.ID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var completed node.Node
	if err := json.NewDecoder(w.Body).Decode(&completed); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !completed.Completed {
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
	qs := queueservicepkg.NewQueueService()
	resource1 := resourcepkg.NewResource("resource-1", 1)
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
	qs := queueservicepkg.NewQueueService()
	created, _ := qs.CreateNode("test-entity")

	// Test successful retrieval
	req := httptest.NewRequest(http.MethodGet, "/nodes/"+created.ID, nil)
	w := httptest.NewRecorder()

	qs.GetNodeHandler(w, req, created.ID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var retrieved node.Node
	if err := json.NewDecoder(w.Body).Decode(&retrieved); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if retrieved.ID != created.ID {
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
	qs := queueservicepkg.NewQueueService()
	qs.CreateNode("entity-1")
	qs.CreateNode("entity-2")

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w := httptest.NewRecorder()

	qs.ListNodesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var nodes []node.Node
	if err := json.NewDecoder(w.Body).Decode(&nodes); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
}

func TestListResourcesHandler(t *testing.T) {
	qs := queueservicepkg.NewQueueService()
	qs.AddResource(resourcepkg.NewResource("resource-1", 5))
	qs.AddResource(resourcepkg.NewResource("resource-2", 3))

	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	w := httptest.NewRecorder()

	qs.ListResourcesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resources []resourcepkg.Resource
	if err := json.NewDecoder(w.Body).Decode(&resources); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}
