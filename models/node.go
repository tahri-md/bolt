package models

import "time"

type Node struct {
	ID                string       `json:"id"`
	Address           string       `json:"address"`
	Port              int          `json:"port"`
	Status            string       `json:"status"`
	LastHeartbeat     time.Time    `json:"last_heartbeat"`
	ReplicationFactor int          `json:"replication_factor"`
	ISReplica         bool         `json:"is_replica"`
	PrimaryNodeID     string       `json:"primary_node_id"`
	Replicas          []*Node      `json:"replicas,omitempty"`
	Cache             *Cache       `json:"cache,omitempty"`
	Metrics           *NodeMetrics `json:"metrics,omitempty"`
}

type NodeMetrics struct {
	MemoryUsed   int64         `json:"memory_used"`
	MemoryLimit  int64         `json:"memory_limit"`
	RequestCount int64         `json:"request_count"`
	AvgLatency   time.Duration `json:"avg_latency"`
	ErrorCount   int64         `json:"error_count"`
}
