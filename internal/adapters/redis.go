package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
	"time"
)

type CacheServer interface {
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Remember(ctx context.Context, key string, value func() interface{}) interface{}
	Get(ctx context.Context, key string) (string, error)
	Pop(ctx context.Context, key string) (string, error)
	Push(ctx context.Context, key string, values ...interface{}) error
	List(ctx context.Context, key string) ([]string, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	DecrBy(ctx context.Context, key string, decrement int64) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	RateLimiter(ctx context.Context, key string, value int, expiration time.Duration) (int64, error)
	CountRateLimiter(ctx context.Context, key string, value int, decrement int, expiration time.Duration) (int64, error)
}

type RedisClient struct {
	Client    *redis.Client
	Available bool
}

var (
	redisClientInstance *RedisClient
	once                sync.Once
)

// Redis returns a singleton Redis client, initializing it only once
func Redis(client *RedisClient) *RedisClient {
	once.Do(func() {
		redisClientInstance = client
		fmt.Println("Reinit")
	})
	return redisClientInstance
}

// Incr increments the value of a key
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// Decr decrements the value of a key
func (r *RedisClient) Decr(ctx context.Context, key string) (int64, error) {
	return r.Client.Decr(ctx, key).Result()
}

// Set sets a value for a given key with an expiration time
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Remember(ctx context.Context, key string, value func() interface{}) interface{} {
	result, err := r.Get(ctx, key)
	if err != nil {
		temp := value()
		return temp
	}
	return result
}

// Get retrieves the value for a given key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Pop pops a value from a list (LPop operation)
func (r *RedisClient) Pop(ctx context.Context, key string) (string, error) {
	return r.Client.LPop(ctx, key).Result()
}

// Push pushes a value to a list (LPush operation)
func (r *RedisClient) Push(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.LPush(ctx, key, values...).Err()
}

// List retrieves all the elements of a list
func (r *RedisClient) List(ctx context.Context, key string) ([]string, error) {
	return r.Client.LRange(ctx, key, 0, -1).Result()
}

// SetNX sets a value to a key only if the key does not exist
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, key, value, expiration).Result()
}

// DecrBy decrements the value of a key by a specified decrement
func (r *RedisClient) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	return r.Client.DecrBy(ctx, key, decrement).Result()
}

// Expire sets an expiration time for a given key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return r.Client.Expire(ctx, key, expiration).Result()
}

// RateLimiter limits the rate of a specific action by decrementing a counter
func (r *RedisClient) RateLimiter(ctx context.Context, key string, value int, expiration time.Duration) (int64, error) {
	_, err := r.Client.Pipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.SetNX(ctx, key, value, expiration).Err(); err != nil {
			return err
		}
		p.Decr(ctx, key)
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Fetch the current value
	return r.Client.Get(ctx, key).Int64()
}

// CountRateLimiter decrements a counter and ensures it does not go below 0
func (r *RedisClient) CountRateLimiter(ctx context.Context, key string, value int, decrement int, expiration time.Duration) (int64, error) {
	// Initialize the key if not exists
	_, err := r.Client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return 0, err
	}

	// Get current value
	currentValStr, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	currentVal, err := strconv.Atoi(currentValStr)
	if err != nil {
		return 0, err
	}

	// Calculate new value
	newValue := currentVal - decrement
	if newValue < 0 {
		return int64(newValue), nil
	}

	// Decrement the key
	_, err = r.Client.DecrBy(ctx, key, int64(decrement)).Result()
	if err != nil {
		return 0, err
	}

	return int64(newValue), nil
}

func RememberWithType[T any](r *RedisClient, ctx context.Context, key string, value func() T) (T, error) {
	// Try to retrieve the value from Redis
	result, err := r.Get(ctx, key)
	if err != nil || result == "" {
		// Generate the value using the provided function
		temp := value()

		// Marshal the value to store it in Redis
		data, marshalErr := json.Marshal(temp)
		if marshalErr != nil {
			return temp, marshalErr
		}

		// Store the marshaled value in Redis
		if setErr := r.Set(ctx, key, string(data), 0); setErr != nil {
			return temp, setErr
		}

		fmt.Println("cache miss")
		return temp, nil
	}

	fmt.Println("cache hit")
	// Unmarshal the result into the generic type T
	var parsed T
	unmarshalErr := json.Unmarshal([]byte(result), &parsed)
	if unmarshalErr != nil {
		var zero T
		return zero, unmarshalErr
	}

	return parsed, nil
}
