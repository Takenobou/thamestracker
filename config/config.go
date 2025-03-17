package config

import (
	"github.com/BurntSushi/toml"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
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
		logger.Logger.Errorf("Failed to load config: %v", err)
		panic(err)
	}
	logger.Logger.Infof("Configuration loaded successfully")
	logger.Logger.Infof("Port of London API URL: %s", AppConfig.URLs.PortOfLondon)
	logger.Logger.Infof("Tower Bridge URL: %s", AppConfig.URLs.TowerBridge)
	logger.Logger.Infof("Redis Address: %s", AppConfig.Redis.Address)
}
