package config

import (
	"log"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
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

func LoadConfig() {
	// Set default values
	AppConfig.Server.Port = 8080
	AppConfig.URLs.PortOfLondon = "https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists"
	AppConfig.URLs.TowerBridge = "https://www.towerbridge.org.uk/lift-times"
	AppConfig.Redis.Address = "localhost:6379"

	// Load configuration from file if it exists
	configPath := "config/config.toml"
	if _, err := os.Stat(configPath); err == nil {
		if _, err := toml.DecodeFile(configPath, &AppConfig); err != nil {
			log.Printf("Error parsing config file: %v", err)
		}
	}

	// Override with environment variables
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			AppConfig.Server.Port = port
		} else {
			log.Printf("Invalid PORT environment variable: %s", portStr)
		}
	}

	if redisAddr := os.Getenv("REDIS_ADDRESS"); redisAddr != "" {
		AppConfig.Redis.Address = redisAddr
	}
}
