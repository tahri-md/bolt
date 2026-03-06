package service

import (
	"bolt/models"
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type CoordinatorService struct {
	mu               sync.RWMutex
	hashRing         *HashRingService
	caches           map[string]*CacheService
	nodes            map[string]*models.Node
	maxCacheSize     int64
	evictionPolicy   string
	heartbeatTimeout time.Duration
}

func NewCoordinatorService(vnodes int, replicationFactor int, maxCacheSize int64, evictionPolicy string) *CoordinatorService {
	return &CoordinatorService{
		hashRing:         NewHashRingService(vnodes, replicationFactor),
		caches:           make(map[string]*CacheService),
		nodes:            make(map[string]*models.Node),
		maxCacheSize:     maxCacheSize,
		evictionPolicy:   evictionPolicy,
		heartbeatTimeout: 15 * time.Second,
	}
}

// --- health monitoring ---

func (c *CoordinatorService) StartHealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Health monitor started")

	for {
		select {
		case <-ticker.C:
			c.checkNodeHealth()
		case <-ctx.Done():
			log.Println("Health monitor stopped")
			return
		}
	}
}

func (c *CoordinatorService) checkNodeHealth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for id, node := range c.nodes {
		if now.Sub(node.LastHeartbeat) > c.heartbeatTimeout {
			if node.Status == "active" {
				node.Status = "inactive"
				log.Printf("Node %s marked as inactive (no heartbeat for %v)", id, c.heartbeatTimeout)
			}
		}
	}
}

func (c *CoordinatorService) UpdateHeartbeat(nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.nodes[nodeID]
	if !exists {
		return errors.New("node not found")
	}

	node.LastHeartbeat = time.Now()
	if node.Status == "inactive" {
		node.Status = "active"
		log.Printf("Node %s is back online", nodeID)
	}

	return nil
}

func (c *CoordinatorService) isNodeActive(nodeID string) bool {
	node, exists := c.nodes[nodeID]
	if !exists {
		return false
	}
	return node.Status == "active"
}

// --- node management ---

func (c *CoordinatorService) AddNode(node *models.Node) error {
	if node == nil || node.ID == "" {
		return errors.New("missing node ID")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[node.ID]; exists {
		return errors.New("node already exists")
	}

	node.Status = "active"
	node.LastHeartbeat = time.Now()
	c.nodes[node.ID] = node
	c.caches[node.ID] = NewCacheService(c.maxCacheSize, c.evictionPolicy)
	c.hashRing.AddNode(node)

	return nil
}

func (c *CoordinatorService) RemoveNode(nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[nodeID]; !exists {
		return errors.New("node not found")
	}

	delete(c.nodes, nodeID)
	delete(c.caches, nodeID)
	c.hashRing.RemoveNode(nodeID)

	return nil
}

// --- cache operations ---

func (c *CoordinatorService) Set(key string, value string, ttl time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes, err := c.hashRing.GetReplicaNodes(key, c.hashRing.hashRing.ReplicationFactor)
	if err != nil {
		return err
	}

	wrote := false
	var lastErr error
	for _, node := range nodes {
		if !c.isNodeActive(node.ID) {
			continue
		}
		cache, exists := c.caches[node.ID]
		if !exists {
			lastErr = errors.New("cache not found for node: " + node.ID)
			continue
		}
		if err := cache.Set(key, value, ttl); err != nil {
			lastErr = err
			continue
		}
		wrote = true
	}

	if !wrote {
		if lastErr != nil {
			return lastErr
		}
		return errors.New("no active nodes available")
	}

	return nil
}

func (c *CoordinatorService) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// try primary first
	primaryNode, err := c.hashRing.GetNode(key)
	if err != nil {
		return "", err
	}

	if c.isNodeActive(primaryNode.ID) {
		cache, exists := c.caches[primaryNode.ID]
		if exists {
			val, err := cache.Get(key)
			if err == nil {
				return val, nil
			}
		}
	}

	// primary failed or inactive — try replicas
	nodes, err := c.hashRing.GetReplicaNodes(key, c.hashRing.hashRing.ReplicationFactor)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		if node.ID == primaryNode.ID {
			continue
		}
		if !c.isNodeActive(node.ID) {
			continue
		}
		cache, exists := c.caches[node.ID]
		if !exists {
			continue
		}
		val, err := cache.Get(key)
		if err == nil {
			return val, nil
		}
	}

	return "", errors.New("key not found on any active node")
}

func (c *CoordinatorService) Delete(key string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes, err := c.hashRing.GetReplicaNodes(key, c.hashRing.hashRing.ReplicationFactor)
	if err != nil {
		return err
	}

	var lastErr error
	for _, node := range nodes {
		if !c.isNodeActive(node.ID) {
			continue
		}
		cache, exists := c.caches[node.ID]
		if !exists {
			lastErr = errors.New("cache not found for node: " + node.ID)
			continue
		}
		if err := cache.Delete(key); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (c *CoordinatorService) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cache := range c.caches {
		cache.Flush()
	}
}

// --- query helpers ---

func (c *CoordinatorService) GetAllNodes() []*models.Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*models.Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		out = append(out, node)
	}
	return out
}

func (c *CoordinatorService) GetNodeForKey(key string) (*models.Node, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.hashRing.GetNode(key)
}

func (c *CoordinatorService) GetNodeMetrics(nodeID string) (*models.CacheMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cache, exists := c.caches[nodeID]
	if !exists {
		return nil, errors.New("node not found")
	}

	return cache.GetMetrics(), nil
}

func (c *CoordinatorService) GetAllMetrics() map[string]*models.CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]*models.CacheMetrics, len(c.caches))
	for nodeID, cache := range c.caches {
		out[nodeID] = cache.GetMetrics()
	}
	return out
}

func (c *CoordinatorService) StartAllTTLCleanup(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for nodeID, cache := range c.caches {
		go func(id string, cs *CacheService) {
			cs.StartTTLCleanup(ctx)
		}(nodeID, cache)
	}
}
