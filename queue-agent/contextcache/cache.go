package contextcache

import (
	"sync"
	"time"

	"queue-common/models"
)

type Snapshot[T any] struct {
	FetchedAt time.Time
	RawJSON   []byte
	Data      T
}

type Cache struct {
	mu sync.RWMutex

	nodesOK bool
	nodes   Snapshot[models.NodesMetricsResponse]

	resOK bool
	res   Snapshot[models.ResourcesSessionMetricsResponse]
}

func New() *Cache { return &Cache{} }

func (c *Cache) SetNodes(s Snapshot[models.NodesMetricsResponse]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes = s
	c.nodesOK = true
}

func (c *Cache) SetResources(s Snapshot[models.ResourcesSessionMetricsResponse]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.res = s
	c.resOK = true
}

func (c *Cache) GetNodes() (Snapshot[models.NodesMetricsResponse], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nodes, c.nodesOK
}

func (c *Cache) GetResources() (Snapshot[models.ResourcesSessionMetricsResponse], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.res, c.resOK
}
