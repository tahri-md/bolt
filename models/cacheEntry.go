package models

import "time"

type CacheEntry struct {
	Key          string        `json:"key"`
	Value        string        `json:"value"`
	CreatedAt    time.Time     `json:"created_at"`
	LastAccessed time.Time     `json:"last_accessed"`
	AccessCount  int           `json:"access_count"`
	TTL          time.Duration `json:"ttl"`
	Checksum     string        `json:"checksum"`
	Size         int           `json:"size"`
}
