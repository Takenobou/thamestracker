package scraper

import (
	"fmt"
	"log"
	"strings"

	"github.com/Takenobou/thamestracker/config"

	"github.com/gocolly/colly"
)

// BridgeLift represents a Tower Bridge lift event
type BridgeLift struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Vessel    string `json:"vessel"`
	Direction string `json:"direction"`
}

// ScrapeBridgeLifts fetches upcoming bridge lift times from all pages
func ScrapeBridgeLifts() ([]BridgeLift, error) {
	baseURL := config.AppConfig.URLs.TowerBridge
	c := colly.NewCollector()
	var lifts []BridgeLift

	// Scrape lift data from the page
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := BridgeLift{
			Date:      e.ChildAttr("td:nth-child(2) time", "datetime"),
			Time:      e.ChildAttr("td:nth-child(3) time", "datetime"),
			Vessel:    strings.TrimSpace(e.ChildText("td:nth-child(4)")),
			Direction: strings.TrimSpace(e.ChildText("td:nth-child(5)")),
		}

		// Format datetime to human-readable format
		if lift.Date != "" {
			lift.Date = lift.Date[:10] // Keep only YYYY-MM-DD part
		}
		if lift.Time != "" {
			lift.Time = lift.Time[11:16] // Extract HH:MM
		}

		lifts = append(lifts, lift)
	})

	// Find and follow pagination links
	c.OnHTML("nav.pager a[title='Current page']", func(e *colly.HTMLElement) {
		nextPage := e.DOM.Parent().Next().Find("a").AttrOr("href", "")
		if nextPage != "" {
			nextURL := fmt.Sprintf("%s%s", baseURL, nextPage)
			log.Println("Scraping next page:", nextURL)
			c.Visit(nextURL) // Recursively visit the next page
		}
	})

	// Start scraping from the first page
	err := c.Visit(baseURL)
	if err != nil {
		log.Println("Error scraping Tower Bridge lifts:", err)
		return nil, err
	}

	c.Wait()

	return lifts, nil
}
