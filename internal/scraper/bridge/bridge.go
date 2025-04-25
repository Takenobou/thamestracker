package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
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
		rawTime := e.ChildAttr("td:nth-child(3) time", "datetime")
		if rawTime == "" {
			logger.Logger.Warnf("Missing datetime for vessel row, skipping")
			return
		}
		tParsed, err := time.Parse(time.RFC3339, rawTime)
		if err != nil {
			logger.Logger.Errorf("Error parsing datetime %s: %v", rawTime, err)
			return
		}
		loc, _ := time.LoadLocation("Europe/London")
		tLondon := tParsed.In(loc)

		lift := models.BridgeLift{
			Date:      tLondon.Format("2006-01-02"),
			Time:      tLondon.Format("15:04"),
			Vessel:    strings.TrimSpace(e.ChildText("td:nth-child(4)")),
			Direction: strings.TrimSpace(e.ChildText("td:nth-child(5)")),
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
