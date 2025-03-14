package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

// Config structure
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
	if _, err := toml.DecodeFile("config/config.toml", &AppConfig); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded successfully ✅")
	log.Printf("Port of London API URL: %s", AppConfig.URLs.PortOfLondon)
	log.Printf("Tower Bridge URL: %s", AppConfig.URLs.TowerBridge)
}
