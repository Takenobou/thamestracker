package bridge

import (
	"fmt"
	"log"
	"strings"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gocolly/colly"
)

// ScrapeBridgeLifts fetches upcoming bridge lift times
func ScrapeBridgeLifts() ([]models.BridgeLift, error) {
	baseURL := config.AppConfig.URLs.TowerBridge
	c := colly.NewCollector()
	var lifts []models.BridgeLift

	// Scrape lift data
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := models.BridgeLift{
			Date:      e.ChildAttr("td:nth-child(2) time", "datetime"),
			Time:      e.ChildAttr("td:nth-child(3) time", "datetime"),
			Vessel:    strings.TrimSpace(e.ChildText("td:nth-child(4)")),
			Direction: strings.TrimSpace(e.ChildText("td:nth-child(5)")),
		}
		// Format Date & Time
		if lift.Date != "" {
			lift.Date = lift.Date[:10] // Keep only YYYY-MM-DD
		}
		if lift.Time != "" {
			lift.Time = lift.Time[11:16] // Extract HH:MM
		}
		lifts = append(lifts, lift)
	})

	// Handle pagination
	c.OnHTML("nav.pager a[title='Current page']", func(e *colly.HTMLElement) {
		nextPage := e.DOM.Parent().Next().Find("a").AttrOr("href", "")
		if nextPage != "" {
			nextURL := fmt.Sprintf("%s%s", baseURL, nextPage)
			log.Println("Scraping next page:", nextURL)
			c.Visit(nextURL)
		}
	})

	// Start scraping
	err := c.Visit(baseURL)
	if err != nil {
		log.Println("Error scraping Tower Bridge lifts:", err)
		return nil, err
	}

	c.Wait()
	return lifts, nil
}
