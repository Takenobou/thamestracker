package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/logger"
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
	logger.Logger.Infof("Saving data to Redis, key: %s", key)
	return r.client.Set(r.ctx, key, jsonData, ttl).Err()
}

// Get retrieves data from Redis.
func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		logger.Logger.Warnf("Redis cache miss, key: %s, error: %v", key, err)
		return errors.New("cache miss")
	}
	logger.Logger.Infof("Cache hit, key: %s", key)
	return json.Unmarshal([]byte(data), dest)
}
