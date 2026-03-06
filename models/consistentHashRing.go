package models

import "sync"

type ConsistentHashRing struct {
	Mu                sync.RWMutex
	Nodes             map[uint64]*Node
	Ring              []uint64
	VirtualNodes      int
	ReplicationFactor int
}
