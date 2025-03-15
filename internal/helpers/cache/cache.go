package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"context"

	"github.com/redis/go-redis/v9"
)

// Cache defines the interface for a cache.
type Cache interface {
	Set(key string, value interface{}, ttl time.Duration) error
	Get(key string, dest interface{}) error
}

// RedisCache is a Redis implementation of the Cache interface.
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache creates a new RedisCache.
func NewRedisCache(addr string) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		ctx: context.Background(),
	}
}

// Set stores data in Redis.
func (r *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fmt.Println("Saving data to Redis:", key)
	return r.client.Set(r.ctx, key, jsonData, ttl).Err()
}

// Get retrieves data from Redis.
func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		fmt.Println("Redis cache miss ❌:", err)
		return errors.New("cache miss")
	}
	fmt.Println("Cache hit ✅, returning data from Redis")
	return json.Unmarshal([]byte(data), dest)
}
