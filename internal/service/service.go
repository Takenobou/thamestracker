package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	keycache "github.com/Takenobou/thamestracker/internal/cache"
	"github.com/Takenobou/thamestracker/internal/config"
	cache "github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/metrics"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// Define BridgeScraper and VesselScraper interfaces

type BridgeScraper interface {
	ScrapeBridgeLifts() ([]models.Event, error)
}

type VesselScraper interface {
	ScrapeVessels(vesselType string) ([]models.Event, error)
}

// Update Service struct to use the interfaces

type Service struct {
	HTTPClient    httpclient.Client
	Cache         cache.Cache
	BridgeScraper BridgeScraper
	VesselScraper VesselScraper
}

// Update NewService to accept the new dependencies
func NewService(httpClient httpclient.Client, cache cache.Cache, bridgeScraper BridgeScraper, vesselScraper VesselScraper) *Service {
	return &Service{
		HTTPClient:    httpClient,
		Cache:         cache,
		BridgeScraper: bridgeScraper,
		VesselScraper: vesselScraper,
	}
}

// singleton redis client for reuse
var redisClientSingleton *redis.Client
var redisOnce sync.Once

func getRedisClient() *redis.Client {
	redisOnce.Do(func() {
		redisClientSingleton = redis.NewClient(&redis.Options{Addr: config.AppConfig.Redis.Address})
	})
	return redisClientSingleton
}

// GetBridgeLifts returns bridge lift events as []Event.
func (s *Service) GetBridgeLifts() ([]models.Event, error) {
	var events []models.Event
	key := keycache.KeyBridgeLifts()
	if err := s.Cache.Get(key, &events); err != nil {
		metrics.CacheMisses.Inc()
		timer := prometheus.NewTimer(metrics.ScrapeDuration.WithLabelValues("bridge"))
		metrics.ScrapeCounter.WithLabelValues("bridge").Inc()
		l, err2 := s.BridgeScraper.ScrapeBridgeLifts()
		timer.ObserveDuration()
		if err2 != nil {
			return nil, err2
		}
		events = l
		if err3 := s.Cache.Set(key, events, 15*time.Minute); err3 != nil {
			logger.Logger.Errorf("Failed to cache bridge_lifts: %v", err3)
			return nil, err3
		}
	} else {
		metrics.CacheHits.Inc()
	}
	return events, nil
}

// GetVessels returns vessel events as []Event.
func (s *Service) GetVessels(vesselType string) ([]models.Event, error) {
	vt := strings.ToLower(vesselType)
	switch vt {
	case "inport", "arrivals", "departures", "forecast", "all":
		// valid
	default:
		return nil, fmt.Errorf("invalid vesselType: %s", vesselType)
	}
	vesselType = vt
	var events []models.Event
	key := keycache.KeyVessels(vesselType)
	if err := s.Cache.Get(key, &events); err != nil {
		metrics.CacheMisses.Inc()
		timer := prometheus.NewTimer(metrics.ScrapeDuration.WithLabelValues("vessels"))
		metrics.ScrapeCounter.WithLabelValues("vessels").Inc()
		data, err2 := s.VesselScraper.ScrapeVessels(vesselType)
		timer.ObserveDuration()
		if err2 != nil {
			return nil, err2
		}
		events = data
		if err3 := s.Cache.Set(key, events, 30*time.Minute); err3 != nil {
			logger.Logger.Errorf("Failed to cache %s: %v", key, err3)
			return nil, err3
		}
	} else {
		metrics.CacheHits.Inc()
	}
	return events, nil
}

// Add caching for filtered vessels by type and location
func (s *Service) GetFilteredVessels(vesselType, location string) ([]models.Event, error) {
	vt := strings.ToLower(vesselType)
	if strings.TrimSpace(location) == "" || vt == "all" {
		return s.GetVessels(vt)
	}
	key := keycache.KeyVesselsByLoc(vt, location)
	events := make([]models.Event, 0)
	if err := s.Cache.Get(key, &events); err == nil {
		metrics.CacheHits.Inc()
		return events, nil
	}
	metrics.CacheMisses.Inc()
	raw, err := s.GetVessels(vt)
	if err != nil {
		return nil, err
	}
	filtered := make([]models.Event, 0)
	for _, e := range raw {
		if strings.EqualFold(e.Location, location) ||
			strings.EqualFold(e.From, location) ||
			strings.EqualFold(e.To, location) {
			filtered = append(filtered, e)
		}
	}
	logger.Logger.Infof("Retrieved filtered events from API, type: %s location: %s, count: %d", vt, location, len(filtered))
	if err := s.Cache.Set(key, filtered, 30*time.Minute); err != nil {
		logger.Logger.Errorf("Failed to cache %s: %v", key, err)
	}
	return filtered, nil
}

// LocationStats holds aggregated stats for a location.
type LocationStats struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	Inport     int    `json:"inport"`
	Arrivals   int    `json:"arrivals"`
	Departures int    `json:"departures"`
	Forecast   int    `json:"forecast"`
	Total      int    `json:"total"`
}

// ListLocations aggregates event counts by location.
func (s *Service) ListLocations() ([]LocationStats, error) {
	events, err := s.GetVessels("all")
	if err != nil {
		return nil, err
	}
	statsMap := make(map[string]*LocationStats)
	for _, e := range events {
		var name string
		switch e.Category {
		case "inport":
			name = e.Location
		case "arrivals":
			name = e.To
		case "departures":
			name = e.From
		case "forecast":
			name = e.To
		default:
			continue
		}
		if name == "" {
			continue
		}
		if _, ok := statsMap[name]; !ok {
			statsMap[name] = &LocationStats{Name: name, Code: ""}
		}
		stat := statsMap[name]
		switch e.Category {
		case "inport":
			stat.Inport++
		case "arrivals":
			stat.Arrivals++
		case "departures":
			stat.Departures++
		case "forecast":
			stat.Forecast++
		}
	}
	var list []LocationStats
	for _, stat := range statsMap {
		stat.Total = stat.Inport + stat.Arrivals + stat.Departures + stat.Forecast
		list = append(list, *stat)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Total > list[j].Total
	})
	return list, nil
}

func (s *Service) HealthCheck(ctx context.Context) error {
	// Ping Redis with context cancellation support
	rdb := getRedisClient()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	// Check external API via HEAD with context
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, config.AppConfig.URLs.PortOfLondon, nil)
	if err != nil {
		return fmt.Errorf("creating HEAD request failed: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("external API HEAD failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("external API returned status %d", resp.StatusCode)
	}
	// also ensure ListLocations works
	if _, err := s.ListLocations(); err != nil {
		return fmt.Errorf("ListLocations health check failed: %w", err)
	}
	return nil
}
