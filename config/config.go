package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Server struct {
		Port int
	}
	URLs struct {
		PortOfLondon string
		TowerBridge  string
	}
	Redis struct {
		Address string
	}
}

var AppConfig Config

func LoadConfig() {
	AppConfig.Server.Port = 8080
	AppConfig.URLs.PortOfLondon = "https://api.portoflondon.example.com"
	AppConfig.URLs.TowerBridge = "https://towerbridge.example.com"
	AppConfig.Redis.Address = "localhost:6379"

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
