package models

import (
	"container/list"
	"sync"
	"time"
)

type Cache struct {
	mu             sync.RWMutex           `json:"-"`
	Store          map[string]*CacheEntry `json:"-"`
	MaxSize        int64                  `json:"max_size"`
	CurrentSize    int64                  `json:"current_size"`
	EvictionPolicy string                 `json:"eviction_policy"`
	AccessOrder    *list.List             `json:"-"`
	FrequencyMap   map[string]int64       `json:"-"`
	TTLManager     *TTLManager            `json:"-"`
	Metrics        *CacheMetrics          `json:"metrics,omitempty"`
}

type CacheMetrics struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	Evictions  int64   `json:"evictions"`
	TotalKeys  int64   `json:"total_keys"`
	MemoryUsed int64   `json:"memory_used"`
	HitRate    float64 `json:"hit_rate"`
}

type TTLManager struct {
	Entries  map[string]time.Time `json:"-"`
	Interval time.Duration        `json:"-"`
}
