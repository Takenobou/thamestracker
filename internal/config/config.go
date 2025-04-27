package config

import (
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
	CircuitBreaker struct {
		MaxFailures    int `toml:"max_failures"`
		CoolOffSeconds int `toml:"cool_off_seconds"`
	} `toml:"circuit_breaker"`
	FallbackCacheSize       int `toml:"fallback_cache_size"`
	FallbackCacheTTLSeconds int `toml:"fallback_cache_ttl_seconds"`
	RequestsPerMin          int `toml:"requests_per_min"`
}

var AppConfig Config

func NewConfig() Config {
	var cfg Config
	// defaults
	cfg.Server.Port = 8080
	cfg.URLs.PortOfLondon = "https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists"
	cfg.URLs.TowerBridge = "https://www.towerbridge.org.uk/lift-times"
	cfg.Redis.Address = "localhost:6379"
	// circuit breaker defaults
	cfg.CircuitBreaker.MaxFailures = 5
	cfg.CircuitBreaker.CoolOffSeconds = 60
	// fallback cache defaults
	cfg.FallbackCacheSize = 1000
	cfg.FallbackCacheTTLSeconds = 3600
	cfg.RequestsPerMin = 60

	// overrides
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("PORT_OF_LONDON"); v != "" {
		cfg.URLs.PortOfLondon = v
	}
	if v := os.Getenv("TOWER_BRIDGE"); v != "" {
		cfg.URLs.TowerBridge = v
	}
	if v := os.Getenv("REDIS_ADDRESS"); v != "" {
		cfg.Redis.Address = v
	}
	if v := os.Getenv("CB_MAX_FAILURES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.CircuitBreaker.MaxFailures = i
		}
	}
	if v := os.Getenv("CB_COOL_OFF"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.CircuitBreaker.CoolOffSeconds = i
		}
	}
	// overrides for fallback cache
	if v := os.Getenv("CACHE_MAX_ENTRIES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.FallbackCacheSize = i
		}
	}
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.FallbackCacheTTLSeconds = i
		}
	}
	if v := os.Getenv("REQUESTS_PER_MIN"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.RequestsPerMin = i
		}
	}

	return cfg
}

// LoadConfig populates the global AppConfig from environment and defaults.
func LoadConfig() {
	AppConfig = NewConfig()
}
