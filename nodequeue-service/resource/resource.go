package resource

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"sync"

	"nodequeue-service/node"
)

// Resource represents a capacity-limited worker pool.
//
// Important invariant:
// - WaitingQueue does NOT consume capacity.
// - Nodes (service queue) DOES consume capacity.
//
// Nodes are typically added to WaitingQueue first, then promoted into Nodes via AllocateWaitingNode.
type Resource struct {
	ID       string `json:"id"`
	Capacity int    `json:"capacity"`
	// Nodes represents the service queue (nodes currently consuming capacity)
	Nodes []*node.Node `json:"nodes"`
	// WaitingQueue represents nodes assigned to this resource but not yet consuming capacity
	WaitingQueue []*node.Node `json:"waiting_queue"`
	mu           sync.RWMutex
}

// IsInService reports whether the given node ID is currently in the service queue.
func (r *Resource) IsInService(nodeID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, n := range r.Nodes {
		if n.ID == nodeID {
			return true
		}
	}
	return false
}

// NewResource constructs a Resource with initialized queues and the provided capacity.
func NewResource(id string, capacity int) *Resource {
	return &Resource{
		ID:           id,
		Capacity:     capacity,
		Nodes:        make([]*node.Node, 0),
		WaitingQueue: make([]*node.Node, 0),
	}
}

// AddNode assigns a node to the resource by placing it into the waiting queue.
// Capacity is enforced when allocating from waiting -> service.
func (r *Resource) AddNode(n *node.Node) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.WaitingQueue = append(r.WaitingQueue, n)
	n.ResourceID = r.ID
	n.AddResourceID(r.ID)
	return true
}

// AllocateWaitingNode promotes a node from the waiting queue into the service queue.
//
// Returns false if:
// - the Resource is already at capacity, or
// - the node is not present in the waiting queue.
func (r *Resource) AllocateWaitingNode(nodeID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Nodes) >= r.Capacity {
		return false
	}

	for i, node := range r.WaitingQueue {
		if node.ID == nodeID {
			// remove the node from the waiting queue
			r.WaitingQueue = append(r.WaitingQueue[:i], r.WaitingQueue[i+1:]...)
			// Add this to allocated queue
			r.Nodes = append(r.Nodes, node)
			return true
		}
	}

	return false
}

// RemoveNode removes a node from the resource, searching both the service queue and waiting queue.
// It returns true if a node was removed.
func (r *Resource) RemoveNode(nodeID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, node := range r.Nodes {
		if node.ID == nodeID {
			r.Nodes = append(r.Nodes[:i], r.Nodes[i+1:]...)
			return true
		}
	}

	for i, node := range r.WaitingQueue {
		if node.ID == nodeID {
			r.WaitingQueue = append(r.WaitingQueue[:i], r.WaitingQueue[i+1:]...)
			return true
		}
	}
	return false
}

// GetNode looks up a node in the resource by ID, searching both the service and waiting queues.
// It returns nil if the node is not present.
func (r *Resource) GetNode(nodeID string) *node.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, node := range r.Nodes {
		if node.ID == nodeID {
			return node
		}
	}
	for _, node := range r.WaitingQueue {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// GetAvailableCapacity returns remaining capacity based on the service queue size.
// Nodes in WaitingQueue do not affect this value.
func (r *Resource) GetAvailableCapacity() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.Capacity - len(r.Nodes)
}

// IsFull reports whether the service queue has reached capacity.
func (r *Resource) IsFull() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Nodes) >= r.Capacity
}

// Util functions for Resource

type resourceConfig struct {
	id       string
	capacity int
}

// loadResources attempts to read resource definitions from a CSV file.
// If the file does not exist (or yields no valid rows), it falls back to defaults.
//
// Expected CSV format: id,capacity (with an optional header row like "Name,Capacity").
func loadResources(fileName string) []resourceConfig {
	resources := make([]resourceConfig, 0)

	configFile, err := os.Open(fileName)
	if err == nil {
		defer configFile.Close()
		reader := csv.NewReader(configFile)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil || len(record) < 2 || record[0] == "Name" {
				continue // skip malformed lines and header
			}
			cap, err := strconv.Atoi(record[1])
			if err != nil {
				continue // skip if capacity field is not integer
			}
			resources = append(resources, resourceConfig{id: record[0], capacity: cap})
		}
	}

	// If file is missing OR produced no valid resources, use defaults.
	if len(resources) == 0 {
		resources = []resourceConfig{
			{id: "Room 1", capacity: 5},
			{id: "Room 2", capacity: 3},
			{id: "Room 3", capacity: 4},
		}
	}
	return resources
}

// LoadResources returns initialized Resource instances based on a CSV config file,
// falling back to built-in defaults when the file is missing or empty.
func LoadResources(fileName string) []*Resource {
	cfgs := loadResources(fileName)
	out := make([]*Resource, 0, len(cfgs))
	for _, c := range cfgs {
		out = append(out, NewResource(c.id, c.capacity))
	}
	return out
}
