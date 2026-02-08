package bridge

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gocolly/colly"
)

// ScrapeBridgeLifts fetches upcoming bridge lift times as unified events.
func ScrapeBridgeLifts() ([]models.Event, error) {
	baseURL := config.AppConfig.URLs.TowerBridge
	if baseURL == "" {
		logger.Logger.Errorf("Tower Bridge URL is missing: set TOWER_BRIDGE environment variable")
		return nil, fmt.Errorf("missing Tower Bridge URL")
	}
	logger.Logger.Infof("Fetching Tower Bridge lifts, url: %s", baseURL)

	c := colly.NewCollector()
	var events []models.Event
	foundPager := false

	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		rawTime := e.ChildAttr("td:nth-child(3) time", "datetime")
		if rawTime == "" {
			logger.Logger.Warnf("Missing datetime for vessel row, skipping")
			return
		}
		tParsed, err := parseBridgeTimestamp(rawTime)
		if err != nil {
			logger.Logger.Errorf("Error parsing datetime %s: %v", rawTime, err)
			return
		}
		vesselName := strings.TrimSpace(e.ChildText("td:nth-child(4)"))
		direction := strings.TrimSpace(e.ChildText("td:nth-child(5)"))
		event := models.Event{
			Timestamp:  tParsed,
			VesselName: vesselName,
			Category:   "bridge",
			Direction:  direction,
			Location:   "Tower Bridge Road, London",
		}
		logger.Logger.Infof("Found lift event: vessel: %s, timestamp: %s, direction: %s",
			vesselName, tParsed.Format(time.RFC3339), direction)
		events = append(events, event)
	})

	// Handle pagination.
	c.OnHTML("nav.pager a[title='Current page']", func(e *colly.HTMLElement) {
		foundPager = true
		nextPage := e.DOM.Parent().Next().Find("a").AttrOr("href", "")
		if nextPage != "" {
			baseParsed, err := url.Parse(baseURL)
			if err != nil {
				return
			}
			nextParsed, err := url.Parse(nextPage)
			if err != nil {
				return
			}
			// If absolute URL, ensure same host
			if nextParsed.IsAbs() && nextParsed.Host != baseParsed.Host {
				logger.Logger.Warnf("Skipping external next page URL: %s", nextParsed)
				return
			}
			// Resolve relative URL
			safeURL := baseParsed.ResolveReference(nextParsed).String()
			logger.Logger.Infof("Scraping next page, url: %s", safeURL)
			c.Visit(safeURL)
		}
	})

	// Start scraping with retry
	if err := utils.Retry(3, 500*time.Millisecond, func() error {
		return c.Visit(baseURL)
	}); err != nil {
		logger.Logger.Errorf("Error scraping Tower Bridge lifts after retries: %v", err)
		return nil, err
	}

	c.Wait()
	if !foundPager {
		logger.Logger.Warnf("Bridge scraper: missing pagination link, structure may have changed")
	}
	logger.Logger.Infof("Retrieved bridge lift events from API, count: %d", len(events))
	return events, nil
}

// BridgeScraperImpl is a concrete implementation of service.BridgeScraper
// that calls ScrapeBridgeLifts.
type BridgeScraperImpl struct{}

func (BridgeScraperImpl) ScrapeBridgeLifts() ([]models.Event, error) {
	return ScrapeBridgeLifts()
}

// parseBridgeTimestamp supports both the current epoch-seconds format and
// older ISO-like values returned by Tower Bridge pages.
func parseBridgeTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}

	if epochSeconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(epochSeconds, 0).In(utils.LondonLocation), nil
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.In(utils.LondonLocation), nil
	}

	legacyISO := strings.TrimSuffix(raw, "Z")
	t, err := time.ParseInLocation("2006-01-02T15:04:05", legacyISO, utils.LondonLocation)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
