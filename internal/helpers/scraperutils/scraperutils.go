package scraperutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gocolly/colly"
)

// ParseBridgeLiftElement extracts a BridgeLift from a table row element.
func ParseBridgeLiftElement(e *colly.HTMLElement) (*models.BridgeLift, error) {
	rawTime := e.ChildAttr("td:nth-child(3) time", "datetime")
	if rawTime == "" {
		return nil, fmt.Errorf("missing datetime attribute")
	}
	tParsed, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing datetime %s: %w", rawTime, err)
	}
	loc, _ := time.LoadLocation("Europe/London")
	tLondon := tParsed.In(loc)

	vessel := strings.TrimSpace(e.ChildText("td:nth-child(4)"))
	direction := strings.TrimSpace(e.ChildText("td:nth-child(5)"))

	return &models.BridgeLift{
		Date:      tLondon.Format("2006-01-02"),
		Time:      tLondon.Format("15:04"),
		Vessel:    vessel,
		Direction: direction,
	}, nil
}
