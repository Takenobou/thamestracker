package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	ics "github.com/arran4/golang-ical"
	"github.com/gofiber/fiber/v2"
)

type APIHandler struct {
	svc *service.Service
}

func NewAPIHandler(svc *service.Service) *APIHandler {
	return &APIHandler{svc: svc}
}

func (h *APIHandler) GetBridgeLifts(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	lifts, err := h.svc.GetBridgeLifts()
	if err != nil {
		logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
	}
	// Apply unique filtering if requested.
	if opts.Unique {
		lifts = utils.FilterUniqueLifts(lifts, 4)
	}
	// Additional location filtering (using vessel name) if specified.
	if opts.Location != "" {
		var filtered []interface{}
		for _, lift := range lifts {
			if strings.Contains(strings.ToLower(lift.Vessel), opts.Location) {
				filtered = append(filtered, lift)
			}
		}
		// Recast to []models.BridgeLift.
		lifts = make([]models.BridgeLift, len(filtered))
		for i, v := range filtered {
			lifts[i] = v.(models.BridgeLift)
		}
	}
	return c.JSON(lifts)
}

func (h *APIHandler) GetShips(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	ships, err := h.svc.GetShips(opts.ShipType)
	if err != nil {
		logger.Logger.Errorf("Error fetching ship data: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve ship data"})
	}
	ships = utils.FilterShips(ships, c)
	return c.JSON(ships)
}

func (h *APIHandler) CalendarHandler(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	now := time.Now()

	// Bridge events in the calendar.
	if opts.EventType == "all" || opts.EventType == "bridge" {
		lifts, err := h.svc.GetBridgeLifts()
		if err != nil {
			logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		} else {
			if opts.Unique {
				lifts = utils.FilterUniqueLifts(lifts, 4)
			}
			for i, lift := range lifts {
				// Filter by location if provided.
				if opts.Location != "" && !strings.Contains(strings.ToLower(lift.Vessel), opts.Location) {
					continue
				}
				start, err := time.Parse("2006-01-02 15:04", lift.Date+" "+lift.Time)
				if err != nil {
					logger.Logger.Errorf("Error parsing bridge lift time for vessel %s: %v", lift.Vessel, err)
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

	// Ship events in the calendar.
	if opts.EventType == "all" || opts.EventType == "ship" {
		ships, err := h.svc.GetShips("inport")
		if err != nil {
			logger.Logger.Errorf("Error fetching ship data: %v", err)
		} else {
			for i, ship := range ships {
				if opts.Location != "" {
					combinedLocation := strings.ToLower(ship.LocationFrom + " " + ship.LocationTo + " " + ship.LocationName)
					if !strings.Contains(combinedLocation, opts.Location) {
						continue
					}
				}
				// Parse original arrival time.
				originalArrival, err := time.Parse("02/01/2006 15:04", ship.Date+" "+ship.Time)
				if err != nil {
					logger.Logger.Errorf("Error parsing ship time for %s: %v", ship.Name, err)
					continue
				}
				// Override the event start to today's date, preserving time-of-day.
				today := now
				start := time.Date(today.Year(), today.Month(), today.Day(), originalArrival.Hour(), originalArrival.Minute(), 0, 0, today.Location())
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
				event.SetDescription(fmt.Sprintf("Arrived: %s | Voyage: %s", ship.Date+" "+ship.Time, ship.VoyageNo))
			}
		}
	}

	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}
