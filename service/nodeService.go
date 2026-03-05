package service

import (
	"bolt/models"
	"errors"
	"sync"
)

type NodeService struct {
	mu    sync.RWMutex
	store map[string]models.Node
}

func NewNodeService() *NodeService {
	return &NodeService{
		store: make(map[string]models.Node),
	}
}

func (s *NodeService) GetAllNodes() ([]models.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Node, 0, len(s.store))
	for _, v := range s.store {
		out = append(out, v)
	}
	return out, nil
}

func (s *NodeService) GetNodeByID(id string) (*models.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.store[id]
	if !ok {
		return nil, errors.New("node not found")
	}
	cp := v
	return &cp, nil
}

func (s *NodeService) CreateNode(node *models.Node) error {
	if node == nil || node.ID == "" {
		return errors.New("missing node ID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.store[node.ID]; exists {
		return errors.New("node already exists")
	}
	s.store[node.ID] = *node
	return nil
}

func (s *NodeService) UpdateNode(node *models.Node) error {
	if node == nil || node.ID == "" {
		return errors.New("missing node ID for update")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.store[node.ID]; !exists {
		return errors.New("node not found")
	}
	s.store[node.ID] = *node
	return nil
}

func (s *NodeService) DeleteNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.store[id]; !exists {
		return errors.New("node not found")
	}
	delete(s.store, id)
	return nil
}
