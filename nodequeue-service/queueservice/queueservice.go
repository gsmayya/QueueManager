package queueservice

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"nodequeue-service/db"
	"nodequeue-service/node"
	"nodequeue-service/resource"
	"nodequeue-service/utils"

	"github.com/google/uuid"
)

// QueueService is the in-memory orchestration layer for nodes and resources.
//
// Concurrency:
// - qs.mu protects the maps and Node state transitions performed here.
// - Resource has its own internal lock for queue operations.
//
// Semantics:
// - Moving/assigning a node to a resource places it into that resource's waiting queue.
// - Allocation (waiting -> service) is where capacity is enforced.
type QueueService struct {
	resources map[string]*resource.Resource
	nodes     map[string]*node.Node
	store     db.Store
	mu        sync.RWMutex
}

// NewQueueService constructs a QueueService with initialized maps.
func NewQueueService() *QueueService {
	return NewQueueServiceWithStore(nil)
}

// NewQueueServiceWithStore constructs a QueueService with an optional persistence store.
// The store is used on a best-effort basis to avoid changing API behavior if the DB is down.
func NewQueueServiceWithStore(store db.Store) *QueueService {
	return &QueueService{
		resources: make(map[string]*resource.Resource),
		nodes:     make(map[string]*node.Node),
		store:     store,
	}
}

func (qs *QueueService) bestEffortPersist(ctx context.Context, op string, fn func(ctx context.Context) error) {
	if qs.store == nil {
		return
	}
	if err := fn(ctx); err != nil {
		log.Printf("[DB] %s failed: %v", op, err)
	}
}

// AddResource registers a Resource by ID, replacing any existing entry with the same ID.
func (qs *QueueService) AddResource(r *resource.Resource) {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	qs.resources[r.ID] = r
}

// CreateNode creates and stores a new node for the provided entity name.
// The node is created unassigned (ResourceID empty) and includes an initial "created" log entry.
func (qs *QueueService) CreateNode(entityName string) (*node.Node, error) {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	node := &node.Node{
		ID:        uuid.New().String(),
		Entity:    &node.Entity{Name: entityName},
		Completed: false,
		CreatedAt: time.Now(),
	}
	node.AddLog("created", "")

	qs.nodes[node.ID] = node

	// Persist audit trail (best-effort).
	ctx := context.Background()
	entityID := uuid.New().String()
	createdAt := node.CreatedAt
	qs.bestEffortPersist(ctx, "PersistNodeCreated", func(ctx context.Context) error {
		return qs.store.PersistNodeCreated(ctx, node.ID, entityID, entityName, createdAt)
	})
	qs.bestEffortPersist(ctx, "InsertNodeLog(created)", func(ctx context.Context) error {
		return qs.store.InsertNodeLog(ctx, node.ID, "created", nil, createdAt)
	})

	return node, nil
}

// MoveNode assigns a node to a target resource.
//
// If the node was already assigned to another resource, it is removed from that resource
// (both waiting and service queues are searched).
//
// The node is always enqueued into the target resource's waiting queue; capacity is not checked here.
func (qs *QueueService) MoveNode(nodeID, targetResourceID string) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	node, exists := qs.nodes[nodeID]
	if !exists {
		return errors.New("node not found")
	}

	if node.Completed {
		return errors.New("cannot move completed node")
	}

	targetResource, exists := qs.resources[targetResourceID]
	if !exists {
		return errors.New("target resource not found")
	}

	// Remove from current resource if it exists
	if node.ResourceID != "" {
		if currentResource, exists := qs.resources[node.ResourceID]; exists {
			currentResource.RemoveNode(nodeID)
		}
	}

	// Assign to target resource (always goes to waiting queue)
	targetResource.AddNode(node)
	node.AddLog("moved_to_waiting_queue", targetResourceID)

	// Persist audit trail (best-effort).
	ctx := context.Background()
	rid := targetResourceID
	qs.bestEffortPersist(ctx, "UpdateNodeResource(move)", func(ctx context.Context) error {
		return qs.store.UpdateNodeResource(ctx, node.ID, &rid)
	})
	qs.bestEffortPersist(ctx, "InsertNodeLog(moved_to_waiting_queue)", func(ctx context.Context) error {
		return qs.store.InsertNodeLog(ctx, node.ID, "moved_to_waiting_queue", &rid, time.Now())
	})

	return nil
}

// AllocateNode promotes a node from its resource waiting queue into the service queue.
//
// Errors include:
// - node/resource not found
// - node not assigned to a resource
// - node already in service queue
// - resource at full capacity
// - node not present in the waiting queue
func (qs *QueueService) AllocateNode(nodeID string) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	node, exists := qs.nodes[nodeID]
	if !exists {
		return errors.New("node not found")
	}

	if node.Completed {
		return errors.New("cannot allocate completed node")
	}

	if node.ResourceID == "" {
		return errors.New("node is not assigned to a resource")
	}

	resource, exists := qs.resources[node.ResourceID]
	if !exists {
		return errors.New("resource not found")
	}

	// Ensure node is currently in the waiting queue, and enforce capacity on promotion to service
	if resource.IsInService(nodeID) {
		return errors.New("node is already in service queue")
	}

	if resource.IsFull() {
		return errors.New("resource is at full capacity")
	}

	if ok := resource.AllocateWaitingNode(nodeID); !ok {
		return errors.New("node is not in waiting queue")
	}

	node.AddLog("moved_to_service_queue", node.ResourceID)

	// Persist audit trail (best-effort).
	ctx := context.Background()
	rid := node.ResourceID
	qs.bestEffortPersist(ctx, "InsertNodeLog(moved_to_service_queue)", func(ctx context.Context) error {
		return qs.store.InsertNodeLog(ctx, node.ID, "moved_to_service_queue", &rid, time.Now())
	})
	return nil
}

// CompleteNode marks a node as completed and removes it from any resource queues.
// Completed nodes cannot be moved or allocated again.
func (qs *QueueService) CompleteNode(nodeID string) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	node, exists := qs.nodes[nodeID]
	if !exists {
		return errors.New("node not found")
	}

	if node.Completed {
		return errors.New("node is already completed")
	}

	node.Completed = true
	node.AddLog("completed", node.ResourceID)

	// Remove from current resource
	if node.ResourceID != "" {
		if resource, exists := qs.resources[node.ResourceID]; exists {
			resource.RemoveNode(nodeID)
		}
		// Persist node completion + clear resource (best-effort).
		ctx := context.Background()
		rid := node.ResourceID
		qs.bestEffortPersist(ctx, "MarkNodeCompleted(true)", func(ctx context.Context) error {
			return qs.store.MarkNodeCompleted(ctx, node.ID, true)
		})
		qs.bestEffortPersist(ctx, "InsertNodeLog(completed)", func(ctx context.Context) error {
			return qs.store.InsertNodeLog(ctx, node.ID, "completed", &rid, time.Now())
		})
		node.ResourceID = ""
	}

	return nil
}

// GetNode returns a node by ID.
func (qs *QueueService) GetNode(nodeID string) (*node.Node, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	node, exists := qs.nodes[nodeID]
	if !exists {
		return nil, errors.New("node not found")
	}

	return node, nil
}

// GetResource returns a resource by ID.
func (qs *QueueService) GetResource(resourceID string) (*resource.Resource, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	resource, exists := qs.resources[resourceID]
	if !exists {
		return nil, errors.New("resource not found")
	}

	return resource, nil
}

// ListResources returns a snapshot slice of all resources currently registered.
func (qs *QueueService) ListResources() []*resource.Resource {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	resources := make([]*resource.Resource, 0, len(qs.resources))
	for _, resource := range qs.resources {
		resources = append(resources, resource)
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID < resources[j].ID
	})
	return resources
}

// ListNodes returns a snapshot slice of all nodes currently stored.
func (qs *QueueService) ListNodes() []*node.Node {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	nodes := make([]*node.Node, 0, len(qs.nodes))
	for _, node := range qs.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Handlers being called from API end point

// CreateNodeHandler handles POST /nodes.
//
// Behavior:
// - Validates payload and creates a node.
// - Optionally assigns it to a resource waiting queue if resource_id is provided.
// - Returns the created node (with its lifecycle log).
func (qs *QueueService) CreateNodeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req node.CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[API] POST /nodes - ERROR: Invalid request body - %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EntityName == "" {
		log.Printf("[API] POST /nodes - ERROR: entity_name is required")
		utils.RespondWithError(w, http.StatusBadRequest, "entity_name is required")
		return
	}

	log.Printf("[API] POST /nodes - Request: entity_name=%s, resource_id=%s", req.EntityName, req.ResourceID)

	node, err := qs.CreateNode(req.EntityName)
	if err != nil {
		log.Printf("[API] POST /nodes - ERROR: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If resource_id is provided, add node to that resource
	if req.ResourceID != "" {
		log.Printf("[API] POST /nodes - Moving node %s to resource %s", node.ID, req.ResourceID)
		if err := qs.MoveNode(node.ID, req.ResourceID); err != nil {
			log.Printf("[API] POST /nodes - ERROR moving node: %v", err)
			// If move fails, still return the created node
			utils.RespondWithJSON(w, http.StatusCreated, node)
			return
		}
		// Refresh node to get updated state
		node, _ = qs.GetNode(node.ID)
	}

	duration := time.Since(startTime)
	log.Printf("[API] POST /nodes - SUCCESS: Created node %s (took %v)", node.ID, duration)
	utils.RespondWithJSON(w, http.StatusCreated, node)
}

// MoveNodeHandler handles POST /nodes/{id}/move.
//
// This assigns the node to the target resource by placing it in the target's waiting queue.
// It does not allocate the node into service; use POST /nodes/{id}/allocate for that.
func (qs *QueueService) MoveNodeHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	startTime := time.Now()
	log.Printf("[API] POST /nodes/%s/move - Request", nodeID)

	var req node.MoveNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[API] POST /nodes/%s/move - ERROR: Invalid request body - %v", nodeID, err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TargetResourceID == "" {
		log.Printf("[API] POST /nodes/%s/move - ERROR: target_resource_id is required", nodeID)
		utils.RespondWithError(w, http.StatusBadRequest, "target_resource_id is required")
		return
	}

	log.Printf("[API] POST /nodes/%s/move - Moving to resource %s", nodeID, req.TargetResourceID)
	if err := qs.MoveNode(nodeID, req.TargetResourceID); err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "node not found" || err.Error() == "target resource not found" {
			statusCode = http.StatusNotFound
		}
		log.Printf("[API] POST /nodes/%s/move - ERROR: %v", nodeID, err)
		utils.RespondWithError(w, statusCode, err.Error())
		return
	}

	duration := time.Since(startTime)
	log.Printf("[API] POST /nodes/%s/move - SUCCESS: Moved to resource %s (took %v)", nodeID, req.TargetResourceID, duration)
	node, _ := qs.GetNode(nodeID)
	utils.RespondWithJSON(w, http.StatusOK, node)
}

// CompleteNodeHandler handles POST /nodes/{id}/complete.
//
// Completion marks a node immutable (no further moves/allocations) and removes it from any queues.
func (qs *QueueService) CompleteNodeHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	startTime := time.Now()
	log.Printf("[API] POST /nodes/%s/complete - Request", nodeID)

	if err := qs.CompleteNode(nodeID); err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "node not found" {
			statusCode = http.StatusNotFound
		}
		log.Printf("[API] POST /nodes/%s/complete - ERROR: %v", nodeID, err)
		utils.RespondWithError(w, statusCode, err.Error())
		return
	}

	duration := time.Since(startTime)
	log.Printf("[API] POST /nodes/%s/complete - SUCCESS: Node completed (took %v)", nodeID, duration)
	node, _ := qs.GetNode(nodeID)
	utils.RespondWithJSON(w, http.StatusOK, node)
}

// AllocateNodeHandler handles POST /nodes/{id}/allocate.
//
// Allocation promotes a node from the assigned resource's waiting queue into the service queue.
// This is the step where resource capacity is enforced.
func (qs *QueueService) AllocateNodeHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	startTime := time.Now()
	log.Printf("[API] POST /nodes/%s/allocate - Request", nodeID)

	if err := qs.AllocateNode(nodeID); err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "node not found" || err.Error() == "resource not found" {
			statusCode = http.StatusNotFound
		}
		log.Printf("[API] POST /nodes/%s/allocate - ERROR: %v", nodeID, err)
		utils.RespondWithError(w, statusCode, err.Error())
		return
	}

	duration := time.Since(startTime)
	log.Printf("[API] POST /nodes/%s/allocate - SUCCESS: Node allocated (took %v)", nodeID, duration)
	node, _ := qs.GetNode(nodeID)
	utils.RespondWithJSON(w, http.StatusOK, node)
}

// GetNodeHandler handles GET /nodes/{id}.
// Returns 404 if the node does not exist.
func (qs *QueueService) GetNodeHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	log.Printf("[API] GET /nodes/%s - Request", nodeID)
	node, err := qs.GetNode(nodeID)
	if err != nil {
		log.Printf("[API] GET /nodes/%s - ERROR: %v", nodeID, err)
		utils.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	log.Printf("[API] GET /nodes/%s - SUCCESS", nodeID)
	utils.RespondWithJSON(w, http.StatusOK, node)
}

// ListNodesHandler handles GET /nodes.
func (qs *QueueService) ListNodesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[API] GET /nodes - Request")
	nodes := qs.ListNodes()
	log.Printf("[API] GET /nodes - SUCCESS: Returning %d nodes", len(nodes))
	utils.RespondWithJSON(w, http.StatusOK, nodes)
}

// ListResourcesHandler handles GET /resources.
func (qs *QueueService) ListResourcesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[API] GET /resources - Request")
	resources := qs.ListResources()
	log.Printf("[API] GET /resources - SUCCESS: Returning %d resources", len(resources))
	utils.RespondWithJSON(w, http.StatusOK, resources)
}
