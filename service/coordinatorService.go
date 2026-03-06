package service

import (
	"bolt/models"
	"context"
	"errors"
	"sync"
	"time"
)

type CoordinatorService struct {
	mu             sync.RWMutex
	hashRing       *HashRingService
	caches         map[string]*CacheService
	nodes          map[string]*models.Node
	maxCacheSize   int64
	evictionPolicy string
}

func NewCoordinatorService(vnodes int, replicationFactor int, maxCacheSize int64, evictionPolicy string) *CoordinatorService {
	return &CoordinatorService{
		hashRing:       NewHashRingService(vnodes, replicationFactor),
		caches:         make(map[string]*CacheService),
		nodes:          make(map[string]*models.Node),
		maxCacheSize:   maxCacheSize,
		evictionPolicy: evictionPolicy,
	}
}

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

func (c *CoordinatorService) Set(key string, value string, ttl time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes, err := c.hashRing.GetReplicaNodes(key, c.hashRing.hashRing.ReplicationFactor)
	if err != nil {
		return err
	}

	var lastErr error
	for _, node := range nodes {
		cache, exists := c.caches[node.ID]
		if !exists {
			lastErr = errors.New("cache not found for node: " + node.ID)
			continue
		}
		if err := cache.Set(key, value, ttl); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (c *CoordinatorService) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// try primary first
	primaryNode, err := c.hashRing.GetNode(key)
	if err != nil {
		return "", err
	}

	cache, exists := c.caches[primaryNode.ID]
	if exists {
		val, err := cache.Get(key)
		if err == nil {
			return val, nil
		}
	}

	// primary failed — try replicas
	nodes, err := c.hashRing.GetReplicaNodes(key, c.hashRing.hashRing.ReplicationFactor)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		if node.ID == primaryNode.ID {
			continue // already tried
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

	return "", errors.New("key not found on any node")
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
