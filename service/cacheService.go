package service

import (
	"bolt/models"
	"container/list"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type CacheService struct {
	mu    sync.RWMutex
	cache models.Cache
}

func NewCacheService(maxSize int64, evictionPolicy string) *CacheService {
	return &CacheService{
		cache: models.Cache{
			Store:          make(map[string]*list.Element),
			MaxSize:        maxSize,
			CurrentSize:    0,
			EvictionPolicy: evictionPolicy,
			AccessOrder:    list.New(),
			FrequencyMap:   make(map[string]int64),
			TTLManager: &models.TTLManager{
				Entries:  make(map[string]time.Time),
				Interval: 1 * time.Second,
			},
			Metrics: &models.CacheMetrics{},
		},
	}
}

func (s *CacheService) StartTTLCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.cache.TTLManager.Interval)
	defer ticker.Stop()

	log.Println("TTL cleanup goroutine started")

	for {
		select {
		case <-ticker.C:
			s.cleanupExpired()
		case <-ctx.Done():
			log.Println("TTL cleanup goroutine stopped")
			return
		}
	}
}

func (s *CacheService) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	expired := make([]string, 0)
	for key, expiry := range s.cache.TTLManager.Entries {
		if now.After(expiry) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		element, exists := s.cache.Store[key]
		if !exists {
			delete(s.cache.TTLManager.Entries, key)
			continue
		}
		s.removeElement(element)
		s.cache.Metrics.Evictions++
	}

	if len(expired) > 0 {
		log.Printf("TTL cleanup: removed %d expired entries", len(expired))
	}
}

func (s *CacheService) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, exists := s.cache.Store[key]
	if !exists {
		s.cache.Metrics.Misses++
		s.updateHitRate()
		return "", errors.New("cache entry not found")
	}

	entry := element.Value.(*models.CacheEntry)

	// lazy TTL check — if expired, remove it
	if expiry, ok := s.cache.TTLManager.Entries[key]; ok && entry.TTL > 0 {
		if time.Now().After(expiry) {
			s.removeElement(element)
			s.cache.Metrics.Misses++
			s.updateHitRate()
			return "", errors.New("cache entry expired")
		}
	}

	if entry.Checksum != "" && computeChecksum(entry.Value) != entry.Checksum {
		s.removeElement(element)
		s.cache.Metrics.Misses++
		s.updateHitRate()
		return "", errors.New("cache entry corrupted: checksum mismatch")
	}

	s.cache.AccessOrder.MoveToFront(element)
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	s.cache.FrequencyMap[key]++
	s.cache.Metrics.Hits++
	s.updateHitRate()

	return entry.Value, nil
}

func (s *CacheService) Set(key string, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if element, exists := s.cache.Store[key]; exists {
		s.removeElement(element)
	}

	newEntry := &models.CacheEntry{
		Key:          key,
		Value:        value,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  0,
		TTL:          ttl,
		Checksum:     computeChecksum(value),
		Size:         len(value),
	}

	for s.cache.CurrentSize+int64(newEntry.Size) > s.cache.MaxSize {
		if s.cache.AccessOrder.Len() == 0 {
			break
		}
		s.evictOne()
	}

	element := s.cache.AccessOrder.PushFront(newEntry)
	s.cache.Store[key] = element
	s.cache.CurrentSize += int64(newEntry.Size)
	s.cache.FrequencyMap[key] = 1
	s.cache.Metrics.TotalKeys++
	s.cache.Metrics.MemoryUsed = s.cache.CurrentSize

	if ttl > 0 {
		s.cache.TTLManager.Entries[key] = time.Now().Add(ttl)
	}

	return nil
}

func (s *CacheService) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, exists := s.cache.Store[key]
	if !exists {
		return errors.New("cache entry not found")
	}

	s.removeElement(element)
	return nil
}

func (s *CacheService) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Store = make(map[string]*list.Element)
	s.cache.AccessOrder.Init()
	s.cache.TTLManager.Entries = make(map[string]time.Time)
	s.cache.FrequencyMap = make(map[string]int64)
	s.cache.CurrentSize = 0
	s.cache.Metrics = &models.CacheMetrics{}
}

func (s *CacheService) GetAllEntries() ([]models.CacheEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.CacheEntry, 0, s.cache.AccessOrder.Len())
	for e := s.cache.AccessOrder.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*models.CacheEntry)
		out = append(out, *entry)
	}
	return out, nil
}

func (s *CacheService) GetMetrics() *models.CacheMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache.Metrics
}

// internal helpers
func (s *CacheService) evictOne() {
	switch s.cache.EvictionPolicy {
	case "LFU":
		s.evictLFU()
	default: // "LRU"
		s.evictLRU()
	}
}

func (s *CacheService) evictLRU() {
	element := s.cache.AccessOrder.Back()
	if element == nil {
		return
	}
	s.removeElement(element)
	s.cache.Metrics.Evictions++
}

func (s *CacheService) evictLFU() {
	var minElement *list.Element
	var minFreq int64 = -1

	for e := s.cache.AccessOrder.Back(); e != nil; e = e.Prev() {
		entry := e.Value.(*models.CacheEntry)
		freq := s.cache.FrequencyMap[entry.Key]

		if minFreq == -1 || freq < minFreq {
			minFreq = freq
			minElement = e
		}
	}

	if minElement != nil {
		s.removeElement(minElement)
		s.cache.Metrics.Evictions++
	}
}

func (s *CacheService) removeElement(element *list.Element) {
	entry := element.Value.(*models.CacheEntry)
	s.cache.AccessOrder.Remove(element)
	delete(s.cache.Store, entry.Key)
	delete(s.cache.TTLManager.Entries, entry.Key)
	delete(s.cache.FrequencyMap, entry.Key)
	s.cache.CurrentSize -= int64(entry.Size)
	s.cache.Metrics.TotalKeys--
	s.cache.Metrics.MemoryUsed = s.cache.CurrentSize
}

func (s *CacheService) updateHitRate() {
	total := s.cache.Metrics.Hits + s.cache.Metrics.Misses
	if total > 0 {
		s.cache.Metrics.HitRate = float64(s.cache.Metrics.Hits) / float64(total)
	}
}

func computeChecksum(value string) string {
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", hash)
}
