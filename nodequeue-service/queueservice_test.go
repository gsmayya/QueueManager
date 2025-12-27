package main

import (
	"testing"
)

func TestNewQueueService(t *testing.T) {
	qs := NewQueueService()

	if qs.resources == nil {
		t.Error("Resources map should be initialized")
	}
	if qs.nodes == nil {
		t.Error("Nodes map should be initialized")
	}
}

func TestQueueService_AddResource(t *testing.T) {
	qs := NewQueueService()
	resource := NewResource("test-resource", 5)

	qs.AddResource(resource)

	retrievedResource, err := qs.GetResource("test-resource")
	if err != nil {
		t.Errorf("Failed to retrieve added resource: %v", err)
	}
	if retrievedResource.ID != "test-resource" {
		t.Errorf("Expected resource ID 'test-resource', got '%s'", retrievedResource.ID)
	}
}

func TestQueueService_CreateNode(t *testing.T) {
	qs := NewQueueService()

	node, err := qs.CreateNode("test-entity")
	if err != nil {
		t.Errorf("Failed to create node: %v", err)
	}

	if node == nil {
		t.Error("Created node should not be nil")
	}
	if node.ID == "" {
		t.Error("Node ID should not be empty")
	}
	if node.Entity == nil {
		t.Error("Node Entity should not be nil")
	}
	if node.Entity.Name != "test-entity" {
		t.Errorf("Expected entity name 'test-entity', got '%s'", node.Entity.Name)
	}
	if node.Completed {
		t.Error("New node should not be completed")
	}
	if node.ResourceID != "" {
		t.Error("New node should not have a resource ID")
	}

	if len(node.Log) == 0 || node.Log[0].Action != "created" {
		t.Error("New node should have a creation log entry")
	}

	// Verify node is stored
	retrievedNode, err := qs.GetNode(node.ID)
	if err != nil {
		t.Errorf("Failed to retrieve created node: %v", err)
	}
	if retrievedNode.ID != node.ID {
		t.Errorf("Retrieved node ID mismatch")
	}
}

func TestQueueService_MoveNode(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 3)
	resource2 := NewResource("resource-2", 2)
	qs.AddResource(resource1)
	qs.AddResource(resource2)

	node, _ := qs.CreateNode("test-entity")

	// Move node to resource1
	err := qs.MoveNode(node.ID, "resource-1")
	if err != nil {
		t.Errorf("Failed to move node: %v", err)
	}

	retrievedNode, _ := qs.GetNode(node.ID)
	if retrievedNode.ResourceID != "resource-1" {
		t.Errorf("Expected ResourceID 'resource-1', got '%s'", retrievedNode.ResourceID)
	}

	// Move node from resource1 to resource2
	err = qs.MoveNode(node.ID, "resource-2")
	if err != nil {
		t.Errorf("Failed to move node to resource2: %v", err)
	}

	retrievedNode, _ = qs.GetNode(node.ID)
	if retrievedNode.ResourceID != "resource-2" {
		t.Errorf("Expected ResourceID 'resource-2', got '%s'", retrievedNode.ResourceID)
	}

	// Verify node was removed from resource1
	if resource1.GetNode(node.ID) != nil {
		t.Error("Node should be removed from resource1")
	}

	// Verify node is in resource2
	if resource2.GetNode(node.ID) == nil {
		t.Error("Node should be in resource2")
	}
}

func TestQueueService_MoveNode_Errors(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 1)
	qs.AddResource(resource1)

	node, _ := qs.CreateNode("test-entity")

	// Try to move non-existent node
	err := qs.MoveNode("non-existent", "resource-1")
	if err == nil {
		t.Error("Should return error for non-existent node")
	}

	// Try to move to non-existent resource
	err = qs.MoveNode(node.ID, "non-existent")
	if err == nil {
		t.Error("Should return error for non-existent resource")
	}
}

func TestQueueService_AllocateNode(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 1)
	qs.AddResource(resource1)

	node, _ := qs.CreateNode("test-entity")
	if err := qs.MoveNode(node.ID, "resource-1"); err != nil {
		t.Fatalf("Failed to move node to resource: %v", err)
	}

	if err := qs.AllocateNode(node.ID); err != nil {
		t.Fatalf("Failed to allocate node: %v", err)
	}

	if resource1.GetAvailableCapacity() != 0 {
		t.Errorf("Expected available capacity 0, got %d", resource1.GetAvailableCapacity())
	}

	retrievedNode, _ := qs.GetNode(node.ID)
	foundServiceLog := false
	for _, entry := range retrievedNode.Log {
		if entry.Action == "moved_to_service_queue" && entry.ResourceID == "resource-1" {
			foundServiceLog = true
			break
		}
	}
	if !foundServiceLog {
		t.Error("Expected node log to include moved_to_service_queue")
	}
}

func TestQueueService_AllocateNode_Errors(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 1)
	qs.AddResource(resource1)

	// Not assigned
	node, _ := qs.CreateNode("test-entity")
	if err := qs.AllocateNode(node.ID); err == nil {
		t.Error("Should return error when allocating unassigned node")
	}

	// Capacity exceeded
	node1, _ := qs.CreateNode("entity-1")
	node2, _ := qs.CreateNode("entity-2")
	qs.MoveNode(node1.ID, "resource-1")
	qs.MoveNode(node2.ID, "resource-1")

	if err := qs.AllocateNode(node1.ID); err != nil {
		t.Fatalf("Expected first allocation to succeed, got %v", err)
	}
	if err := qs.AllocateNode(node2.ID); err == nil {
		t.Error("Should return error when allocating over capacity")
	}
}

func TestQueueService_CompleteNode(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 3)
	qs.AddResource(resource1)

	node, _ := qs.CreateNode("test-entity")
	qs.MoveNode(node.ID, "resource-1")

	// Complete the node
	err := qs.CompleteNode(node.ID)
	if err != nil {
		t.Errorf("Failed to complete node: %v", err)
	}

	retrievedNode, _ := qs.GetNode(node.ID)
	if !retrievedNode.Completed {
		t.Error("Node should be marked as completed")
	}
	if retrievedNode.ResourceID != "" {
		t.Error("Completed node should not have a ResourceID")
	}

	foundCompletedLog := false
	for _, entry := range retrievedNode.Log {
		if entry.Action == "completed" {
			foundCompletedLog = true
			break
		}
	}
	if !foundCompletedLog {
		t.Error("Expected node log to include completed")
	}

	// Verify node was removed from resource
	if resource1.GetNode(node.ID) != nil {
		t.Error("Completed node should be removed from resource")
	}
}

func TestQueueService_CompleteNode_Errors(t *testing.T) {
	qs := NewQueueService()

	// Try to complete non-existent node
	err := qs.CompleteNode("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent node")
	}

	// Try to complete already completed node
	node, _ := qs.CreateNode("test-entity")
	qs.CompleteNode(node.ID)

	err = qs.CompleteNode(node.ID)
	if err == nil {
		t.Error("Should return error for already completed node")
	}
}

func TestQueueService_MoveCompletedNode(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 3)
	qs.AddResource(resource1)

	node, _ := qs.CreateNode("test-entity")
	qs.CompleteNode(node.ID)

	// Try to move completed node
	err := qs.MoveNode(node.ID, "resource-1")
	if err == nil {
		t.Error("Should not be able to move completed node")
	}
}

func TestQueueService_GetNode(t *testing.T) {
	qs := NewQueueService()
	node, _ := qs.CreateNode("test-entity")

	retrievedNode, err := qs.GetNode(node.ID)
	if err != nil {
		t.Errorf("Failed to get node: %v", err)
	}
	if retrievedNode.ID != node.ID {
		t.Errorf("Node ID mismatch")
	}

	// Try to get non-existent node
	_, err = qs.GetNode("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent node")
	}
}

func TestQueueService_GetResource(t *testing.T) {
	qs := NewQueueService()
	resource := NewResource("test-resource", 5)
	qs.AddResource(resource)

	retrievedResource, err := qs.GetResource("test-resource")
	if err != nil {
		t.Errorf("Failed to get resource: %v", err)
	}
	if retrievedResource.ID != "test-resource" {
		t.Errorf("Resource ID mismatch")
	}

	// Try to get non-existent resource
	_, err = qs.GetResource("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent resource")
	}
}

func TestQueueService_ListResources(t *testing.T) {
	qs := NewQueueService()
	resource1 := NewResource("resource-1", 5)
	resource2 := NewResource("resource-2", 3)
	qs.AddResource(resource1)
	qs.AddResource(resource2)

	resources := qs.ListResources()
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}

func TestQueueService_ListNodes(t *testing.T) {
	qs := NewQueueService()
	qs.CreateNode("entity-1")
	qs.CreateNode("entity-2")
	qs.CreateNode("entity-3")

	nodes := qs.ListNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}
}
