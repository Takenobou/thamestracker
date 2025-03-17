package bridge

import (
	"fmt"
	"strings"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gocolly/colly"
)

// ScrapeBridgeLifts fetches upcoming bridge lift times.
func ScrapeBridgeLifts() ([]models.BridgeLift, error) {
	baseURL := config.AppConfig.URLs.TowerBridge
	if baseURL == "" {
		logger.Logger.Errorf("Tower Bridge URL is missing from config")
		return nil, fmt.Errorf("missing Tower Bridge URL")
	}
	logger.Logger.Infof("Fetching Tower Bridge lifts, url: %s", baseURL)

	c := colly.NewCollector()
	var lifts []models.BridgeLift

	// Scrape lift data.
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := models.BridgeLift{
			Date:      e.ChildAttr("td:nth-child(2) time", "datetime"),
			Time:      e.ChildAttr("td:nth-child(3) time", "datetime"),
			Vessel:    strings.TrimSpace(e.ChildText("td:nth-child(4)")),
			Direction: strings.TrimSpace(e.ChildText("td:nth-child(5)")),
		}

		// Format Date & Time.
		if lift.Date != "" {
			lift.Date = lift.Date[:10] // Keep only YYYY-MM-DD.
		} else {
			logger.Logger.Warnf("Missing date in a bridge lift row")
		}
		if lift.Time != "" {
			lift.Time = lift.Time[11:16] // Extract HH:MM.
		} else {
			logger.Logger.Warnf("Missing time for vessel: %s", lift.Vessel)
		}

		logger.Logger.Infof("Found lift: vessel: %s, date: %s, time: %s, direction: %s",
			lift.Vessel, lift.Date, lift.Time, lift.Direction)
		lifts = append(lifts, lift)
	})

	// Handle pagination.
	c.OnHTML("nav.pager a[title='Current page']", func(e *colly.HTMLElement) {
		nextPage := e.DOM.Parent().Next().Find("a").AttrOr("href", "")
		if nextPage != "" {
			nextURL := fmt.Sprintf("%s%s", baseURL, nextPage)
			logger.Logger.Infof("Scraping next page, url: %s", nextURL)
			c.Visit(nextURL)
		}
	})

	// Start scraping.
	err := c.Visit(baseURL)
	if err != nil {
		logger.Logger.Errorf("Error scraping Tower Bridge lifts: %v", err)
		return nil, err
	}

	c.Wait()
	logger.Logger.Infof("Retrieved bridge lifts from API, count: %d", len(lifts))
	return lifts, nil
}
