package config

import "os"

var (
	PortOfLondonURL = os.Getenv("PORT_OF_LONDON_URL")
	TowerBridgeURL  = os.Getenv("TOWER_BRIDGE_URL")
	RedisAddr       = os.Getenv("REDIS_ADDR")
)
