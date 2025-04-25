package test

import (
	"os"
	"testing"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Set environment variables
	err := os.Setenv("PORT", "9090")
	assert.NoError(t, err)
	err = os.Setenv("PORT_OF_LONDON", "http://fake.london")
	assert.NoError(t, err)
	err = os.Setenv("TOWER_BRIDGE", "http://fake.bridge")
	assert.NoError(t, err)
	err = os.Setenv("REDIS_ADDRESS", "redis://localhost:6380")
	assert.NoError(t, err)

	// Load config from env
	config.LoadConfig()

	assert.Equal(t, 9090, config.AppConfig.Server.Port)
	assert.Equal(t, "http://fake.london", config.AppConfig.URLs.PortOfLondon)
	assert.Equal(t, "http://fake.bridge", config.AppConfig.URLs.TowerBridge)
	assert.Equal(t, "redis://localhost:6380", config.AppConfig.Redis.Address)
}
