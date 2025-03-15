package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	shipScraper "github.com/Takenobou/thamestracker/internal/scraper/ships"
	ics "github.com/arran4/golang-ical"
	"github.com/gofiber/fiber/v2"
)

// CalendarHandler generates a dynamic ICS feed based on query parameters.
func CalendarHandler(c *fiber.Ctx) error {
	eventTypeFilter := c.Query("eventType", "all")
	locationFilter := c.Query("location", "")
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	now := time.Now()

	if eventTypeFilter == "all" || eventTypeFilter == "bridge" {
		lifts, err := bridgeScraper.ScrapeBridgeLifts()
		if err != nil {
			log.Println("Error scraping bridge lifts:", err)
		} else {
			uniqueLifts := FilterUniqueLifts(lifts, 4)
			for i, lift := range uniqueLifts {
				// Optionally filter bridge events by location based on vessel name
				if locationFilter != "" && !strings.Contains(strings.ToLower(lift.Vessel), strings.ToLower(locationFilter)) {
					continue
				}
				start, err := time.Parse("2006-01-02 15:04", lift.Date+" "+lift.Time)
				if err != nil {
					log.Printf("Error parsing bridge lift time for %s: %v", lift.Vessel, err)
					continue
				}
				end := start.Add(10 * time.Minute)
				eventID := fmt.Sprintf("bridge-%d@thamestracker", i)
				event := cal.AddEvent(eventID)
				event.SetCreatedTime(now)
				event.SetDtStampTime(now)
				event.SetModifiedAt(now)
				event.SetStartAt(start)
				event.SetEndAt(end)
				event.SetSummary(fmt.Sprintf("Bridge Lift: %s", lift.Vessel))
				event.SetLocation("Tower Bridge")
				event.SetDescription(fmt.Sprintf("Direction: %s", lift.Direction))
			}
		}
	}

	if eventTypeFilter == "all" || eventTypeFilter == "ship" {
		ships, err := shipScraper.ScrapeShips("inport")
		if err != nil {
			log.Println("Error scraping ship data:", err)
		} else {
			// You can also apply location filtering directly here if needed.
			for i, ship := range ships {
				if locationFilter != "" {
					combinedLocation := strings.ToLower(ship.LocationFrom + " " + ship.LocationTo + " " + ship.LocationName)
					if !strings.Contains(combinedLocation, strings.ToLower(locationFilter)) {
						continue
					}
				}
				start, err := time.Parse("02/01/2006 15:04", ship.Date+" "+ship.Time)
				if err != nil {
					log.Printf("Error parsing ship time for %s: %v", ship.Name, err)
					continue
				}
				end := start.Add(15 * time.Minute)
				eventID := fmt.Sprintf("ship-%d@thamestracker", i)
				event := cal.AddEvent(eventID)
				event.SetCreatedTime(now)
				event.SetDtStampTime(now)
				event.SetModifiedAt(now)
				event.SetStartAt(start)
				event.SetEndAt(end)
				event.SetSummary(fmt.Sprintf("Ship In Port: %s", ship.Name))
				event.SetLocation("Port")
				event.SetDescription(fmt.Sprintf("Voyage: %s", ship.VoyageNo))
			}
		}
	}

	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}
