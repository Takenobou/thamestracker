package service

import (
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/models"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	vesselScraper "github.com/Takenobou/thamestracker/internal/scraper/vessels"
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
	if err := s.Cache.Get("bridge_lifts", &lifts); err != nil {
		l, err := s.getBridgeLiftsFromScraper()
		if err != nil {
			return nil, err
		}
		lifts = l
		s.Cache.Set("bridge_lifts", lifts, 15*time.Minute)
	}
	return lifts, nil
}

func (s *Service) getBridgeLiftsFromScraper() ([]models.BridgeLift, error) {
	// Separate helper if needed; here we call the scraper directly.
	return bridgeScraper.ScrapeBridgeLifts() // Assuming bridge lifts remain in the bridge package.
}

func (s *Service) GetVessels(vesselType string) ([]models.Vessel, error) {
	var vessels []models.Vessel
	var cacheKey string
	if vesselType == "all" {
		cacheKey = "all_vessels"
	} else {
		cacheKey = "vessels_" + strings.ToLower(vesselType)
	}
	if err := s.Cache.Get(cacheKey, &vessels); err != nil {
		sData, err := vesselScraper.ScrapeVessels(vesselType)
		if err != nil {
			return nil, err
		}
		vessels = sData
		s.Cache.Set(cacheKey, vessels, 30*time.Minute)
	}
	return vessels, nil
}
