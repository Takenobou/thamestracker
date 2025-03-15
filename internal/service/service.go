package service

import (
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/models"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	shipScraper "github.com/Takenobou/thamestracker/internal/scraper/ships"
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
		l, err := bridgeScraper.ScrapeBridgeLifts()
		if err != nil {
			return nil, err
		}
		lifts = l
		s.Cache.Set("bridge_lifts", lifts, 15*time.Minute)
	}
	return lifts, nil
}

func (s *Service) GetShips(shipType string) ([]models.Ship, error) {
	var ships []models.Ship
	var cacheKey string
	if shipType == "all" {
		cacheKey = "all_ships"
	} else {
		cacheKey = "ships_" + strings.ToLower(shipType)
	}
	if err := s.Cache.Get(cacheKey, &ships); err != nil {
		sData, err := shipScraper.ScrapeShips(shipType)
		if err != nil {
			return nil, err
		}
		ships = sData
		s.Cache.Set(cacheKey, ships, 30*time.Minute)
	}
	return ships, nil
}
