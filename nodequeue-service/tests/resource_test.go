package tests

import (
	"testing"

	"nodequeue-service/node"
	"nodequeue-service/resource"
)

func TestNewResource(t *testing.T) {
	resource := resource.NewResource("test-resource", 5)

	if resource.ID != "test-resource" {
		t.Errorf("Expected ID 'test-resource', got '%s'", resource.ID)
	}

	if resource.Capacity != 5 {
		t.Errorf("Expected capacity 5, got %d", resource.Capacity)
	}

	if len(resource.Nodes) != 0 {
		t.Errorf("Expected empty nodes slice, got %d nodes", len(resource.Nodes))
	}

	if len(resource.WaitingQueue) != 0 {
		t.Errorf("Expected empty waiting queue, got %d nodes", len(resource.WaitingQueue))
	}
}

func TestResource_AddNode(t *testing.T) {
	resource := resource.NewResource("test-resource", 2)
	node1 := &node.Node{ID: "node-1", Entity: &node.Entity{Name: "entity-1"}}
	node2 := &node.Node{ID: "node-2", Entity: &node.Entity{Name: "entity-2"}}
	node3 := &node.Node{ID: "node-3", Entity: &node.Entity{Name: "entity-3"}}

	// Add first node - should go to waiting queue
	if !resource.AddNode(node1) {
		t.Error("Failed to add first node")
	}
	if len(resource.WaitingQueue) != 1 {
		t.Errorf("Expected 1 waiting node, got %d", len(resource.WaitingQueue))
	}
	if node1.ResourceID != resource.ID {
		t.Errorf("Expected ResourceID '%s', got '%s'", resource.ID, node1.ResourceID)
	}

	// Add second node - should go to waiting queue
	if !resource.AddNode(node2) {
		t.Error("Failed to add second node")
	}
	if len(resource.WaitingQueue) != 2 {
		t.Errorf("Expected 2 waiting nodes, got %d", len(resource.WaitingQueue))
	}

	// Add third node - should still succeed (waiting queue is not capacity-limited)
	if !resource.AddNode(node3) {
		t.Error("Failed to add third node to waiting queue")
	}
	if len(resource.WaitingQueue) != 3 {
		t.Errorf("Expected 3 waiting nodes, got %d", len(resource.WaitingQueue))
	}

	// Service queue should still be empty until allocation
	if len(resource.Nodes) != 0 {
		t.Errorf("Expected 0 service nodes, got %d", len(resource.Nodes))
	}
}

func TestResource_RemoveNode(t *testing.T) {
	resource := resource.NewResource("test-resource", 5)
	node1 := &node.Node{ID: "node-1", Entity: &node.Entity{Name: "entity-1"}}
	node2 := &node.Node{ID: "node-2", Entity: &node.Entity{Name: "entity-2"}}

	resource.AddNode(node1)
	resource.AddNode(node2)

	// Remove existing node
	if !resource.RemoveNode("node-1") {
		t.Error("Failed to remove existing node")
	}
	if len(resource.WaitingQueue) != 1 {
		t.Errorf("Expected 1 waiting node, got %d", len(resource.WaitingQueue))
	}
	if resource.WaitingQueue[0].ID != "node-2" {
		t.Errorf("Expected remaining node to be 'node-2', got '%s'", resource.WaitingQueue[0].ID)
	}

	// Try to remove non-existent node
	if resource.RemoveNode("non-existent") {
		t.Error("Should not be able to remove non-existent node")
	}
	if len(resource.WaitingQueue) != 1 {
		t.Errorf("Expected 1 waiting node, got %d", len(resource.WaitingQueue))
	}
}

func TestResource_GetNode(t *testing.T) {
	resource := resource.NewResource("test-resource", 5)
	node1 := &node.Node{ID: "node-1", Entity: &node.Entity{Name: "entity-1"}}
	node2 := &node.Node{ID: "node-2", Entity: &node.Entity{Name: "entity-2"}}

	resource.AddNode(node1)
	resource.AddNode(node2)

	// Get existing node
	foundNode := resource.GetNode("node-1")
	if foundNode == nil {
		t.Error("Failed to get existing node")
	}
	if foundNode.ID != "node-1" {
		t.Errorf("Expected node ID 'node-1', got '%s'", foundNode.ID)
	}

	// Get non-existent node
	notFoundNode := resource.GetNode("non-existent")
	if notFoundNode != nil {
		t.Error("Should not find non-existent node")
	}
}

func TestResource_GetAvailableCapacity(t *testing.T) {
	resource := resource.NewResource("test-resource", 5)

	if resource.GetAvailableCapacity() != 5 {
		t.Errorf("Expected available capacity 5, got %d", resource.GetAvailableCapacity())
	}

	node1 := &node.Node{ID: "node-1", Entity: &node.Entity{Name: "entity-1"}}
	resource.AddNode(node1)

	// Adding to waiting queue does not consume capacity
	if resource.GetAvailableCapacity() != 5 {
		t.Errorf("Expected available capacity 5, got %d", resource.GetAvailableCapacity())
	}

	// Allocating to service queue consumes capacity
	if !resource.AllocateWaitingNode(node1.ID) {
		t.Error("Failed to allocate waiting node into service queue")
	}
	if resource.GetAvailableCapacity() != 4 {
		t.Errorf("Expected available capacity 4, got %d", resource.GetAvailableCapacity())
	}
}

func TestResource_IsFull(t *testing.T) {
	resource := resource.NewResource("test-resource", 2)

	if resource.IsFull() {
		t.Error("Resource should not be full initially")
	}

	node1 := &node.Node{ID: "node-1", Entity: &node.Entity{Name: "entity-1"}}
	node2 := &node.Node{ID: "node-2", Entity: &node.Entity{Name: "entity-2"}}

	resource.AddNode(node1)
	if resource.IsFull() {
		t.Error("Resource should not be full with 1 node waiting")
	}

	resource.AddNode(node2)
	if resource.IsFull() {
		t.Error("Resource should still not be full with 2 nodes waiting")
	}

	// Allocate nodes into service queue, consuming capacity
	resource.AllocateWaitingNode(node1.ID)
	if resource.IsFull() {
		t.Error("Resource should not be full with 1 node in service")
	}

	resource.AllocateWaitingNode(node2.ID)
	if !resource.IsFull() {
		t.Error("Resource should be full with 2 nodes in service")
	}
}
