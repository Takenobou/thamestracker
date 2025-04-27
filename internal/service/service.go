package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
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

func (s *Service) GetBridgeLifts() ([]models.BridgeLift, error) {
	var lifts []models.BridgeLift
	const key = "bridge_lifts"
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
	cacheKey := "vessels_" + vesselType
	if vesselType == "all" {
		cacheKey = "all_vessels"
	}
	if err := s.Cache.Get(cacheKey, &vessels); err != nil {
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
		if err3 := s.Cache.Set(cacheKey, vessels, 30*time.Minute); err3 != nil {
			logger.Logger.Errorf("Failed to cache %s: %v", cacheKey, err3)
			return nil, err3
		}
	} else {
		// cache hit
		metrics.CacheHits.Inc()
	}
	return vessels, nil
}

func (s *Service) HealthCheck() error {
	// Ping Redis
	rdb := redis.NewClient(&redis.Options{Addr: config.AppConfig.Redis.Address})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	// Check Port of London API with GET (HEAD not supported by client interface)
	resp, err := s.HTTPClient.Get(config.AppConfig.URLs.PortOfLondon)
	if err != nil {
		return fmt.Errorf("external API GET failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("external API returned status %d", resp.StatusCode)
	}
	return nil
}
