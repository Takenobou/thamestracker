package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Takenobou/thamestracker/config"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var redisClient = redis.NewClient(&redis.Options{
	Addr: config.AppConfig.Redis.Address,
})

// Default cache expiration time
const DefaultTTL = time.Hour

// SetCache stores data in Redis for the given key.
func SetCache(key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	fmt.Println("Saving data to Redis:", key)
	return redisClient.Set(ctx, key, jsonData, ttl).Err()
}

// GetCache retrieves data from Redis.
func GetCache(key string, dest interface{}) error {
	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		fmt.Println("Redis cache miss ❌:", err)
		return errors.New("cache miss")
	}

	fmt.Println("Cache hit ✅, returning data from Redis")
	return json.Unmarshal([]byte(data), dest)
}
