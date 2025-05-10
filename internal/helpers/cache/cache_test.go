package cache

import (
	"os"
	"testing"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitLogger()
	os.Exit(m.Run())
}

func TestRedisCache_SetGet(t *testing.T) {
	// use in‐memory Redis
	srv, err := miniredis.Run()
	assert.NoError(t, err)
	defer srv.Close()

	c := NewRedisCache(srv.Addr())
	key := "test_key"
	value := map[string]string{"foo": "bar"}
	ttl := 5 * time.Second

	// Set value in cache.
	err = c.Set(key, value, ttl)
	assert.NoError(t, err)

	// Get value from cache.
	var result map[string]string
	err = c.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// Wait for TTL expiration in miniredis (fast-forward time).
	srv.FastForward(ttl + 1*time.Second)
	err = c.Get(key, &result)
	assert.Error(t, err, "expected cache miss after TTL expiration")
}
