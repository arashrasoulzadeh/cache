package pkg

import (
	"cacher/internal/adapters"
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync/atomic"
	"time"
)

type cache struct {
	hitStats         statsMap
	missStats        statsMap
	hitLatency       uint64 // Stores the cumulative latency for hits
	hitCount         uint64 // Tracks the total number of hits
	statsTimer       *time.Ticker
	statsTimerStop   chan bool
	RecordStatistics bool
	Cache            adapters.Cache
}

type Cache interface {
	Wrap(ctx context.Context, key string, value func() interface{}) interface{}
	KeyStatistics(ctx context.Context, key string) (map[string]uint64, error)
	Statistics(ctx context.Context) map[string]map[string]uint64
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	AverageHitLatency(ctx context.Context) float64
}

func (c *cache) Wrap(ctx context.Context, key string, value func() interface{}) interface{} {
	if cachedValue, err := c.Get(ctx, key); err == nil && cachedValue != nil {
		return cachedValue
	}

	// Simulate a cache miss
	result := value()
	_ = c.Set(ctx, key, result)
	return result
}

func (c *cache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now() // Start tracking latency

	data, err := c.Cache.Get(ctx, key)
	if data == nil && err != nil {
		c.miss(key)
	} else {
		c.hit(key)

		// Update hit latency
		latency := uint64(time.Since(start).Microseconds()) // Convert duration to microseconds
		atomic.AddUint64(&c.hitLatency, latency)
		atomic.AddUint64(&c.hitCount, 1)
	}

	return data, err
}

func (c *cache) Set(ctx context.Context, key string, value interface{}) error {
	return c.Cache.Set(ctx, key, value)
}

func (c *cache) KeyStatistics(ctx context.Context, key string) (map[string]uint64, error) {
	hitCount := c.hitStats.get(key)
	missCount := c.missStats.get(key)
	if hitCount == 0 && missCount == 0 {
		return nil, errors.New("no statistics available for the given key")
	}

	return map[string]uint64{
		"hits":   hitCount,
		"misses": missCount,
	}, nil
}

func (c *cache) Statistics(ctx context.Context) map[string]map[string]uint64 {
	hits := c.hitStats.getAll()
	misses := c.missStats.getAll()

	stats := make(map[string]map[string]uint64)
	for key, hitCount := range hits {
		if stats[key] == nil {
			stats[key] = map[string]uint64{}
		}
		stats[key]["hits"] = hitCount
	}
	for key, missCount := range misses {
		if stats[key] == nil {
			stats[key] = map[string]uint64{}
		}
		stats[key]["misses"] = missCount
	}

	return stats
}

// AverageHitLatency calculates the average latency for cache hits in microseconds.
func (c *cache) AverageHitLatency(ctx context.Context) float64 {
	hitCount := atomic.LoadUint64(&c.hitCount)
	if hitCount == 0 {
		return 0.0
	}
	totalLatency := atomic.LoadUint64(&c.hitLatency)
	return float64(totalLatency) / float64(hitCount)
}

func NewCache(recordStatistics bool) Cache {

	redisClient := adapters.Redis(&adapters.RedisClient{Client: redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})})
	c := &cache{
		hitStats:         newStatsMap(),
		missStats:        newStatsMap(),
		statsTimer:       time.NewTicker(1 * time.Second),
		statsTimerStop:   make(chan bool),
		RecordStatistics: recordStatistics,
		Cache:            adapters.NewCache(redisClient),
	}

	go func() {
		for {
			select {
			case <-c.statsTimer.C:
				fmt.Println("Periodic stats update:", c.Statistics(context.Background()))
				fmt.Printf("Average Hit Latency: %.2fµs\n", c.AverageHitLatency(context.Background()))
			case <-c.statsTimerStop:
				fmt.Println("Ticker stopped")
				c.statsTimer.Stop()
				return
			}
		}
	}()

	return c
}

func WrapType[T any](ctx context.Context, key string, cache Cache, value func() T) T {
	result := cache.Wrap(ctx, key, func() interface{} {
		return value()
	})
	return result.(T)
}
