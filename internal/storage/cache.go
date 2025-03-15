package storage

import (
	"time"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
)

var CacheClient = cache.NewRedisCache(config.AppConfig.Redis.Address)

const DefaultTTL = time.Hour

func SetCache(key string, value interface{}, ttl time.Duration) error {
	return CacheClient.Set(key, value, ttl)
}

func GetCache(key string, dest interface{}) error {
	return CacheClient.Get(key, dest)
}
