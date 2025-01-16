package pkg

import (
	"sync"
	"sync/atomic"
)

type statsMap struct {
	data  map[string]*uint64
	mutex sync.RWMutex
}

func newStatsMap() statsMap {
	return statsMap{
		data: make(map[string]*uint64),
	}
}

func (sm *statsMap) increment(key string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.data[key]; !exists {
		var initial uint64 = 0
		sm.data[key] = &initial
	}
	atomic.AddUint64(sm.data[key], 1)
}

func (sm *statsMap) get(key string) uint64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if value, exists := sm.data[key]; exists {
		return atomic.LoadUint64(value)
	}
	return 0
}

func (sm *statsMap) getAll() map[string]uint64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	copy := make(map[string]uint64)
	for key, value := range sm.data {
		copy[key] = atomic.LoadUint64(value)
	}
	return copy
}

func (c *cache) hit(key string) {
	if c.RecordStatistics {
		c.hitStats.increment(key)
	}
}

func (c *cache) miss(key string) {
	if c.RecordStatistics {
		c.missStats.increment(key)
	}
}
