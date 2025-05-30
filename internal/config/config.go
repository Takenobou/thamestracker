package config

import (
	"os"
	"strconv"
	"strings"
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
	CircuitBreaker struct {
		MaxFailures    int
		CoolOffSeconds int
	}
	FallbackCacheSize       int
	FallbackCacheTTLSeconds int
	RequestsPerMin          int
	MetricsPublic           bool

	// Bridge filter config
	BridgeFilterPercentile float64 // e.g. 0.10
	BridgeFilterMaxCount   int     // e.g. 8
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
	// metrics endpoint protection default
	cfg.MetricsPublic = false
	// bridge filter defaults
	cfg.BridgeFilterPercentile = 0.10
	cfg.BridgeFilterMaxCount = 8

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
	// optional metrics public flag
	if mp := os.Getenv("METRICS_PUBLIC"); mp != "" {
		cfg.MetricsPublic = strings.EqualFold(mp, "true")
	}
	// bridge filter overrides
	if v := os.Getenv("BRIDGE_FILTER_PERCENTILE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.BridgeFilterPercentile = f
		}
	}
	if v := os.Getenv("BRIDGE_FILTER_MAX_COUNT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.BridgeFilterMaxCount = i
		}
	}

	return cfg
}

// LoadConfig populates the global AppConfig from environment and defaults.
func LoadConfig() {
	AppConfig = NewConfig()
}
