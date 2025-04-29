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
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	vesselScraper "github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	HTTPClient httpclient.Client
	Cache      cache.Cache
}

func NewService(httpClient httpclient.Client, cache cache.Cache) *Service {
	return &Service{
		HTTPClient: httpClient,
		Cache:      cache,
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

func (s *Service) GetBridgeLifts() ([]models.BridgeLift, error) {
	var lifts []models.BridgeLift
	// use centralized cache key
	key := keycache.KeyBridgeLifts()
	// try cache
	if err := s.Cache.Get(key, &lifts); err != nil {
		// cache miss
		metrics.CacheMisses.Inc()
		// record scrape metrics
		timer := prometheus.NewTimer(metrics.ScrapeDuration.WithLabelValues("bridge"))
		metrics.ScrapeCounter.WithLabelValues("bridge").Inc()
		l, err2 := s.getBridgeLiftsFromScraper()
		timer.ObserveDuration()
		if err2 != nil {
			return nil, err2
		}
		lifts = l
		if err3 := s.Cache.Set(key, lifts, 15*time.Minute); err3 != nil {
			logger.Logger.Errorf("Failed to cache bridge_lifts: %v", err3)
			return nil, err3
		}
	} else {
		// cache hit
		metrics.CacheHits.Inc()
	}
	return lifts, nil
}

func (s *Service) getBridgeLiftsFromScraper() ([]models.BridgeLift, error) {
	// Separate helper if needed; here we call the scraper directly.
	return bridgeScraper.ScrapeBridgeLifts()
}

func (s *Service) GetVessels(vesselType string) ([]models.Vessel, error) {
	// Validate vesselType to prevent cache-key injection
	vt := strings.ToLower(vesselType)
	switch vt {
	case "inport", "arrivals", "departures", "forecast", "all":
		// valid
	default:
		return nil, fmt.Errorf("invalid vesselType: %s", vesselType)
	}
	vesselType = vt
	var vessels []models.Vessel
	// cache key for vessels
	key := keycache.KeyVessels(vesselType)
	if err := s.Cache.Get(key, &vessels); err != nil {
		// cache miss
		metrics.CacheMisses.Inc()
		// record scrape metrics
		timer := prometheus.NewTimer(metrics.ScrapeDuration.WithLabelValues("vessels"))
		metrics.ScrapeCounter.WithLabelValues("vessels").Inc()
		data, err2 := vesselScraper.ScrapeVessels(vesselType)
		timer.ObserveDuration()
		if err2 != nil {
			return nil, err2
		}
		vessels = data
		if err3 := s.Cache.Set(key, vessels, 30*time.Minute); err3 != nil {
			logger.Logger.Errorf("Failed to cache %s: %v", key, err3)
			return nil, err3
		}
	} else {
		// cache hit
		metrics.CacheHits.Inc()
	}
	return vessels, nil
}

// Add caching for filtered vessels by type and location
func (s *Service) GetFilteredVessels(vesselType, location string) ([]models.Vessel, error) {
	vt := strings.ToLower(vesselType)
	// Fast-path: no location filter OR type==all → just return the raw list
	if strings.TrimSpace(location) == "" || vt == "all" {
		return s.GetVessels(vt)
	}
	// composite cache key for filtered vessels
	key := keycache.KeyVesselsByLoc(vt, location)
	// initialize slice to avoid nil
	vessels := make([]models.Vessel, 0)
	if err := s.Cache.Get(key, &vessels); err == nil {
		metrics.CacheHits.Inc()
		return vessels, nil
	}
	metrics.CacheMisses.Inc()
	// fetch raw list
	raw, err := s.GetVessels(vt)
	if err != nil {
		return nil, err
	}
	// filter by location based on type
	for _, v := range raw {
		switch vt {
		case "inport":
			if strings.EqualFold(v.LocationName, location) {
				vessels = append(vessels, v)
			}
		case "arrivals", "forecast":
			if strings.EqualFold(v.LocationTo, location) {
				vessels = append(vessels, v)
			}
		case "departures":
			if strings.EqualFold(v.LocationFrom, location) {
				vessels = append(vessels, v)
			}
		}
	}
	// log retrieval before caching
	logger.Logger.Infof("Retrieved filtered vessels from API, type: %s location: %s, count: %d", vt, location, len(vessels))
	// cache filtered results
	if err := s.Cache.Set(key, vessels, 30*time.Minute); err != nil {
		logger.Logger.Errorf("Failed to cache %s: %v", key, err)
	}
	return vessels, nil
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

// ListLocations aggregates vessel counts by location.
func (s *Service) ListLocations() ([]LocationStats, error) {
	vessels, err := s.GetVessels("all")
	if err != nil {
		return nil, err
	}
	statsMap := make(map[string]*LocationStats)
	for _, v := range vessels {
		var name string
		// choose location field based on vessel type
		switch v.Type {
		case "inport":
			name = v.LocationName
		case "arrivals":
			name = v.LocationTo
		case "departures":
			name = v.LocationFrom
		case "forecast":
			name = v.LocationTo
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
		switch v.Type {
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
	// compute totals and assemble slice
	var list []LocationStats
	for _, stat := range statsMap {
		stat.Total = stat.Inport + stat.Arrivals + stat.Departures + stat.Forecast
		list = append(list, *stat)
	}
	// sort by total desc
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
