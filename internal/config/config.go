package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server struct {
		Port int `toml:"port"`
	} `toml:"server"`
	URLs struct {
		PortOfLondon string `toml:"port_of_london"`
		TowerBridge  string `toml:"tower_bridge"`
	} `toml:"urls"`
	Redis struct {
		Address string `toml:"address"`
	} `toml:"redis"`
}

var AppConfig Config
var exitFunc = os.Exit

func SetExitFunc(f func(int)) {
	exitFunc = f
}

func LoadConfig() {
	// Set default values
	AppConfig.Server.Port = 8080
	// Removed default URLs: enforce critical env-only settings
	// AppConfig.URLs.PortOfLondon = "https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists"
	// AppConfig.URLs.TowerBridge = "https://www.towerbridge.org.uk/lift-times"
	AppConfig.Redis.Address = "localhost:6379"

	// Override with environment variables
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			AppConfig.Server.Port = port
		}
	}
	if poLondon := os.Getenv("PORT_OF_LONDON"); poLondon != "" {
		AppConfig.URLs.PortOfLondon = poLondon
	}
	if tb := os.Getenv("TOWER_BRIDGE"); tb != "" {
		AppConfig.URLs.TowerBridge = tb
	}
	if redisAddr := os.Getenv("REDIS_ADDRESS"); redisAddr != "" {
		AppConfig.Redis.Address = redisAddr
	}

	// Fail fast if critical env variables are missing
	if AppConfig.URLs.PortOfLondon == "" {
		fmt.Fprintln(os.Stderr, "error: PORT_OF_LONDON is required")
		exitFunc(1)
	}
	if AppConfig.URLs.TowerBridge == "" {
		fmt.Fprintln(os.Stderr, "error: TOWER_BRIDGE is required")
		exitFunc(1)
	}
}
