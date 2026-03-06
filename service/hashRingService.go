package service

import (
	"bolt/models"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"
)

type HashRingService struct {
	hashRing *models.ConsistentHashRing
}

func NewHashRingService(virtualNodes int, replicationFactor int) *HashRingService {
	return &HashRingService{
		hashRing: &models.ConsistentHashRing{
			Nodes:             make(map[uint64]*models.Node),
			Ring:              []uint64{},
			VirtualNodes:      virtualNodes,
			ReplicationFactor: replicationFactor,
			Mu:                sync.RWMutex{},
		},
	}
}

func (s *HashRingService) AddNode(node *models.Node) {
	s.hashRing.Mu.Lock()
	defer s.hashRing.Mu.Unlock()
	for i := 0; i < s.hashRing.VirtualNodes; i++ {
		virtualNodeID := fmt.Sprintf("%s#%d", node.ID, i)
		hash := s.hash(virtualNodeID)
		s.hashRing.Nodes[hash] = node
		s.hashRing.Ring = append(s.hashRing.Ring, hash)
	}
	slices.Sort(s.hashRing.Ring)
}

func (s *HashRingService) RemoveNode(nodeID string) {
	s.hashRing.Mu.Lock()
	defer s.hashRing.Mu.Unlock()

	toRemove := make(map[uint64]bool)
	for i := 0; i < s.hashRing.VirtualNodes; i++ {
		virtualNodeID := fmt.Sprintf("%s#%d", nodeID, i)
		hash := s.hash(virtualNodeID)
		toRemove[hash] = true
		delete(s.hashRing.Nodes, hash)
	}

	newRing := make([]uint64, 0, len(s.hashRing.Ring)-len(toRemove))
	for _, h := range s.hashRing.Ring {
		if !toRemove[h] {
			newRing = append(newRing, h)
		}
	}
	s.hashRing.Ring = newRing
}
func (s *HashRingService) GetNode(key string) (*models.Node, error) {
	s.hashRing.Mu.RLock()
	defer s.hashRing.Mu.RUnlock()

	if len(s.hashRing.Ring) == 0 {
		return nil, errors.New("no nodes in the ring")
	}

	hash := s.hash(key)

	// binary search for first position >= hash
	idx := sort.Search(len(s.hashRing.Ring), func(i int) bool {
		return s.hashRing.Ring[i] >= hash
	})

	// wrap around if past the end
	if idx >= len(s.hashRing.Ring) {
		idx = 0
	}

	node := s.hashRing.Nodes[s.hashRing.Ring[idx]]
	return node, nil
}

func (s *HashRingService) GetReplicaNodes(key string, count int) ([]*models.Node, error) {
	s.hashRing.Mu.RLock()
	defer s.hashRing.Mu.RUnlock()

	if len(s.hashRing.Ring) == 0 {
		return nil, errors.New("no nodes in the ring")
	}

	hash := s.hash(key)

	idx := sort.Search(len(s.hashRing.Ring), func(i int) bool {
		return s.hashRing.Ring[i] >= hash
	})

	if idx >= len(s.hashRing.Ring) {
		idx = 0
	}

	// walk clockwise, collect unique physical nodes
	seen := make(map[string]bool)
	nodes := make([]*models.Node, 0, count)

	for i := 0; i < len(s.hashRing.Ring) && len(nodes) < count; i++ {
		pos := (idx + i) % len(s.hashRing.Ring)
		node := s.hashRing.Nodes[s.hashRing.Ring[pos]]

		if !seen[node.ID] {
			seen[node.ID] = true
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func (s *HashRingService) GetRingSize() int {
	s.hashRing.Mu.RLock()
	defer s.hashRing.Mu.RUnlock()
	return len(s.hashRing.Ring)
}

func (s *HashRingService) hash(key string) uint64 {
	h := sha1.Sum([]byte(key))
	return binary.BigEndian.Uint64(h[:8])
}
