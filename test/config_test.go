package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Takenobou/thamestracker/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "configtest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a "config" subdirectory.
	configDir := filepath.Join(tmpDir, "config")
	err = os.Mkdir(configDir, 0755)
	assert.NoError(t, err)

	// Create a temporary config file.
	configFilePath := filepath.Join(configDir, "config.toml")
	testConfig := `
[server]
port = 8080

[urls]
port_of_london = "http://fake.london"
tower_bridge = "http://fake.bridge"

[redis]
address = "localhost:6379"
`
	err = os.WriteFile(configFilePath, []byte(testConfig), 0644)
	assert.NoError(t, err)

	// Change working directory to the temporary directory.
	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer os.Chdir(originalWD)

	// Call LoadConfig (which expects to find config/config.toml).
	config.LoadConfig()

	// Verify that the configuration values were loaded correctly.
	assert.Equal(t, 8080, config.AppConfig.Server.Port)
	assert.Equal(t, "http://fake.london", config.AppConfig.URLs.PortOfLondon)
	assert.Equal(t, "http://fake.bridge", config.AppConfig.URLs.TowerBridge)
	assert.Equal(t, "localhost:6379", config.AppConfig.Redis.Address)
}
