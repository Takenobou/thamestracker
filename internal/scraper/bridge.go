package scraper

import (
	"log"

	"github.com/gocolly/colly"
)

// BridgeLift represents a Tower Bridge lift event
type BridgeLift struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Vessel    string `json:"vessel"`
	Direction string `json:"direction"`
}

// ScrapeBridgeLifts fetches upcoming bridge lift times
func ScrapeBridgeLifts() ([]BridgeLift, error) {
	c := colly.NewCollector()
	var lifts []BridgeLift

	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := BridgeLift{
			Date:      e.ChildAttr("td:nth-child(2) time", "datetime"), // Extract datetime attribute for accuracy
			Time:      e.ChildAttr("td:nth-child(3) time", "datetime"), // Extract time in ISO format
			Vessel:    e.ChildText("td:nth-child(4)"),                  // Vessel name is plain text
			Direction: e.ChildText("td:nth-child(5)"),                  // Direction is plain text
		}

		// Convert ISO 8601 datetime to human-readable format if needed
		if lift.Date != "" {
			lift.Date = lift.Date[:10] // Keep only YYYY-MM-DD part
		}
		if lift.Time != "" {
			lift.Time = lift.Time[11:16] // Extract HH:MM
		}

		lifts = append(lifts, lift)
	})

	err := c.Visit("https://www.towerbridge.org.uk/lift-times")
	if err != nil {
		log.Println("Error scraping Tower Bridge lifts:", err)
		return nil, err
	}

	return lifts, nil
}
