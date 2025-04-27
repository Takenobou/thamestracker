package test

import (
	"os"
	"testing"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Set environment variables for valid config
	err := os.Setenv("PORT", "9090")
	assert.NoError(t, err)
	err = os.Setenv("PORT_OF_LONDON", "http://fake.london")
	assert.NoError(t, err)
	err = os.Setenv("TOWER_BRIDGE", "http://fake.bridge")
	assert.NoError(t, err)
	err = os.Setenv("REDIS_ADDRESS", "redis://localhost:6380")
	assert.NoError(t, err)

	// Instantiate config
	cfg := config.NewConfig()
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "http://fake.london", cfg.URLs.PortOfLondon)
	assert.Equal(t, "http://fake.bridge", cfg.URLs.TowerBridge)
	assert.Equal(t, "redis://localhost:6380", cfg.Redis.Address)
}
