package service

import (
	"bolt/models"
	"errors"
	"sync"
)

type CacheEntryService struct {
	mu    sync.RWMutex
	store map[string]models.CacheEntry
}

func NewCacheEntryService() *CacheEntryService {
	return &CacheEntryService{
		store: make(map[string]models.CacheEntry),
	}
}

func (s *CacheEntryService) GetAllCacheEntries() ([]models.CacheEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.CacheEntry, 0, len(s.store))
	for _, v := range s.store {
		out = append(out, v)
	}
	return out, nil
}

func (s *CacheEntryService) GetCacheEntryByKey(key string) (*models.CacheEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.store[key]
	if !ok {
		return nil, errors.New("cache entry not found")
	}
	cp := v
	return &cp, nil
}

func (s *CacheEntryService) CreateCacheEntry(entry *models.CacheEntry) error {
	if entry == nil || entry.Key == "" {
		return errors.New("missing cache entry key")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.store[entry.Key]; exists {
		return errors.New("cache entry already exists")
	}
	s.store[entry.Key] = *entry
	return nil
}

func (s *CacheEntryService) UpdateCacheEntry(entry *models.CacheEntry) error {
	if entry == nil || entry.Key == "" {
		return errors.New("missing cache entry key for update")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.store[entry.Key]; !exists {
		return errors.New("cache entry not found")
	}
	s.store[entry.Key] = *entry
	return nil
}

func (s *CacheEntryService) DeleteCacheEntry(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.store, key)
	return nil
}

func (s *CacheEntryService) FlushCacheEntry() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store = make(map[string]models.CacheEntry)
	return nil
}
