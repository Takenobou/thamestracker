package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	shipScraper "github.com/Takenobou/thamestracker/internal/scraper/ships"
	"github.com/Takenobou/thamestracker/internal/storage"
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

	// Bridge events (with caching)
	if eventTypeFilter == "all" || eventTypeFilter == "bridge" {
		var lifts []models.BridgeLift
		// Attempt to load cached lifts
		if err := storage.GetCache("bridge_lifts", &lifts); err != nil {
			// Cache miss, so scrape fresh data
			l, err := bridgeScraper.ScrapeBridgeLifts()
			if err != nil {
				log.Println("Error scraping bridge lifts:", err)
			} else {
				lifts = l
				storage.SetCache("bridge_lifts", lifts, 15*time.Minute)
			}
		}
		if len(lifts) > 0 {
			uniqueLifts := FilterUniqueLifts(lifts, 4)
			for i, lift := range uniqueLifts {
				// Optionally filter by vessel name if a locationFilter is provided
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
				event.SetSummary(fmt.Sprintf("Tower Bridge Lift: %s", lift.Vessel))
				event.SetDescription(fmt.Sprintf("Direction: %s", lift.Direction))
				event.SetLocation("Tower Bridge\n222 Tower Bridge Road, London, SE1 2UP, England")
			}
		}
	}

	// Ship events (with caching)
	if eventTypeFilter == "all" || eventTypeFilter == "ship" {
		var ships []models.Ship
		// Attempt to load cached ships
		if err := storage.GetCache("ships_in_port", &ships); err != nil {
			s, err := shipScraper.ScrapeShips("inport")
			if err != nil {
				log.Println("Error scraping ship data:", err)
			} else {
				ships = s
				storage.SetCache("ships_in_port", ships, 30*time.Minute)
			}
		}
		if len(ships) > 0 {
			for i, ship := range ships {
				// Apply location filter if provided
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

				shipLocation := ship.LocationName
				if shipLocation == "" {
					shipLocation = strings.TrimSpace(ship.LocationFrom + " " + ship.LocationTo)
				}
				event.SetLocation(shipLocation)
				event.SetDescription(fmt.Sprintf("Voyage: %s", ship.VoyageNo))
			}
		}
	}

	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}
