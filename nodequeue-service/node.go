package main

import (
	"sync"
	"time"
)

// Node is the unit of work managed by the queue.
//
// A Node has a lifecycle:
// - created
// - assigned to a Resource (enqueued into that Resource's waiting queue)
// - allocated into the Resource's service queue (consumes capacity)
// - completed (removed from resource queues, no further moves/allocations allowed)
//
// All state transitions are recorded in Log.
type Node struct {
	ID     string  `json:"id"`
	Entity *Entity `json:"entity"`
	//TODO: Fix this to be current resource
	ResourceID string    `json:"resource_id,omitempty"`
	Completed  bool      `json:"completed"`
	CreatedAt  time.Time `json:"created_at"`
	resources  []*Resource
	Log        []NodeLog `json:"log"`
	mu         sync.RWMutex
}

func (n *Node) AddResource(resource *Resource) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	// Add list of resources this node has gone through
	n.resources = append(n.resources, resource)
	return true
}

// addLog appends a lifecycle event to the node log.
// It is not concurrency-safe on its own; callers should ensure appropriate external locking.
func (n *Node) addLog(action, resourceID string) {
	n.Log = append(n.Log, NodeLog{
		Action:     action,
		ResourceID: resourceID,
		Timestamp:  time.Now(),
	})
}

// CreateNodeRequest is the request payload for POST /nodes.
//
// If ResourceID is provided, the newly created node is immediately assigned to that resource's
// waiting queue (via MoveNode).
type CreateNodeRequest struct {
	EntityName string `json:"entity_name"`
	ResourceID string `json:"resource_id,omitempty"` // Optional: add to resource immediately
}

// MoveNodeRequest is the request payload for POST /nodes/{id}/move.
type MoveNodeRequest struct {
	TargetResourceID string `json:"target_resource_id"`
}

// ErrorResponse is a consistent JSON error envelope returned by handlers in this service.
type ErrorResponse struct {
	Error string `json:"error"`
}

// NodeLog records an action taken on a node (with optional Resource context) and when it occurred.
//
// Action values are intentionally simple strings to keep the API stable and human-readable.
type NodeLog struct {
	Action     string    `json:"action"`
	ResourceID string    `json:"resource_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
