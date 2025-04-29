package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/metrics"
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
func NewRedisCache(addr string) Cache {
	r := &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		ctx: context.Background(),
	}
	return &CompositeCache{
		redis:    r,
		fallback: newFallbackCache(),
	}
}

// Set stores data in Redis.
func (r *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	logger.Logger.Infof("Saving data to Redis, key: %s", key)
	if err := r.client.Set(r.ctx, key, jsonData, ttl).Err(); err != nil {
		metrics.RedisErrorsTotal.Inc()
		logger.Logger.Errorf("Redis SET error (key=%s): %v", key, err)
		return err
	}
	return nil
}

// Get retrieves data from Redis.
func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		metrics.RedisErrorsTotal.Inc()
		logger.Logger.Warnf("Redis GET error (key=%s): %v", key, err)
		return errors.New("cache miss")
	}
	logger.Logger.Infof("Cache hit, key: %s", key)
	return json.Unmarshal([]byte(data), dest)
}

// entry stores a cached value with expiration.
type entry struct {
	data      []byte
	expiresAt time.Time
	freq      int
}

// fallbackCache is an in-memory LFU cache with TTL.
type fallbackCache struct {
	mu    sync.Mutex
	size  int
	ttl   time.Duration
	items map[string]*entry
}

func newFallbackCache() *fallbackCache {
	cfg := config.AppConfig
	fc := &fallbackCache{
		size:  cfg.FallbackCacheSize,
		ttl:   time.Duration(cfg.FallbackCacheTTLSeconds) * time.Second,
		items: make(map[string]*entry, cfg.FallbackCacheSize),
	}
	// start background purge of expired entries if ttl positive
	if fc.ttl > 0 {
		go fc.startPurge()
	}
	return fc
}

// startPurge periodically deletes expired entries every ttl/2
func (f *fallbackCache) startPurge() {
	ticker := time.NewTicker(f.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		f.mu.Lock()
		now := time.Now()
		for k, e := range f.items {
			if now.After(e.expiresAt) {
				delete(f.items, k)
			}
		}
		f.mu.Unlock()
	}
}

// Set stores in fallback and also evicts LFU when full.
func (f *fallbackCache) Set(key string, value interface{}, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.items) >= f.size {
		// evict least frequently used
		var evictKey string
		minFreq := int(^uint(0) >> 1)
		for k, e := range f.items {
			if e.freq < minFreq {
				minFreq = e.freq
				evictKey = k
			}
		}
		delete(f.items, evictKey)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.items[key] = &entry{data: data, expiresAt: time.Now().Add(f.ttl), freq: 1}
	return nil
}

// Get retrieves from fallback; removes expired entries.
func (f *fallbackCache) Get(key string, dest interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		delete(f.items, key)
		return errors.New("cache miss")
	}
	e.freq++
	return json.Unmarshal(e.data, dest)
}

// CompositeCache wraps Redis and fallback.
type CompositeCache struct {
	redis    *RedisCache
	fallback *fallbackCache
}

// Set tries Redis then fallback.
func (c *CompositeCache) Set(key string, value interface{}, ttl time.Duration) error {
	if err := c.redis.Set(key, value, ttl); err != nil {
		// Redis down, fallback
		return c.fallback.Set(key, value, ttl)
	}
	return nil
}

// Get tries Redis then fallback.
func (c *CompositeCache) Get(key string, dest interface{}) error {
	if err := c.redis.Get(key, dest); err == nil {
		return nil
	}
	// Redis miss or error, try fallback
	return c.fallback.Get(key, dest)
}
