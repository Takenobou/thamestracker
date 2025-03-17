package test

import (
	"testing"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/stretchr/testify/assert"
)

func TestRedisCache_SetGet(t *testing.T) {
	// Note: This test requires Redis running on localhost:6379.
	c := cache.NewRedisCache("localhost:6379")
	key := "test_key"
	value := map[string]string{"foo": "bar"}
	ttl := 5 * time.Second

	// Set value in cache.
	err := c.Set(key, value, ttl)
	assert.NoError(t, err)

	// Get value from cache.
	var result map[string]string
	err = c.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// Wait for TTL expiration.
	time.Sleep(ttl + 1*time.Second)
	err = c.Get(key, &result)
	assert.Error(t, err, "expected cache miss after TTL expiration")
}
