package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
	"github.com/Takenobou/thamestracker/internal/models"
	ics "github.com/arran4/golang-ical"
	"github.com/gofiber/fiber/v2"
)

type ServiceInterface interface {
	GetBridgeLifts() ([]models.BridgeLift, error)
	GetVessels(vesselType string) ([]models.Vessel, error)
	HealthCheck() error
}

type APIHandler struct {
	svc ServiceInterface
}

func NewAPIHandler(svc ServiceInterface) *APIHandler {
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
	// Additional name filtering (using vessel name) if specified.
	if name := strings.ToLower(c.Query("name", "")); name != "" {
		var filtered []models.BridgeLift
		for _, lift := range lifts {
			if strings.Contains(strings.ToLower(lift.Vessel), name) {
				filtered = append(filtered, lift)
			}
		}
		lifts = filtered
	}
	return c.JSON(lifts)
}

func (h *APIHandler) GetVessels(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	vessels, err := h.svc.GetVessels(opts.VesselType)
	if err != nil {
		logger.Logger.Errorf("Error fetching vessel data: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve vessel data"})
	}
	// Apply JSON endpoint filters: name, location, nationality, after, before
	vessels = utils.FilterVessels(vessels, c)
	if opts.Unique {
		vessels = utils.FilterUniqueVessels(vessels)
	}
	return c.JSON(vessels)
}

func (h *APIHandler) CalendarHandler(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	// determine if any vessel-specific filters are applied
	hasVesselFilter := opts.Unique || opts.VesselType != "all" ||
		c.Query("name", "") != "" || c.Query("location", "") != "" ||
		c.Query("nationality", "") != "" || c.Query("after", "") != "" ||
		c.Query("before", "") != ""

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	now := time.Now()

	// Bridge events in the calendar (only when no vessel filters)
	if !hasVesselFilter && (opts.EventType == "all" || opts.EventType == "bridge") {
		lifts, err := h.svc.GetBridgeLifts()
		if err != nil {
			logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		} else {
			if opts.Unique {
				lifts = utils.FilterUniqueLifts(lifts, 4)
			}
			for i, lift := range lifts {
				// Filter by name if provided.
				if name := strings.ToLower(c.Query("name", "")); name != "" && !strings.Contains(strings.ToLower(lift.Vessel), name) {
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
				event.SetLocation("Tower Bridge\\n222 Tower Bridge Road, London, SE1 2UP, England")
			}
		}
	}

	// Vessel events in the calendar.
	if opts.EventType == "all" || opts.EventType == "vessel" {
		// Retrieve with requested vessel type
		vessels, err := h.svc.GetVessels(opts.VesselType)
		if err != nil {
			logger.Logger.Errorf("Error fetching vessel data: %v", err)
		} else {
			// Apply calendar filters to match JSON behavior
			vessels = utils.FilterVessels(vessels, c)
			if opts.Unique {
				vessels = utils.FilterUniqueVessels(vessels)
			}
			for i, vessel := range vessels {
				eventID := fmt.Sprintf("vessel-%d@thamestracker", i)
				event := cal.AddEvent(eventID)
				event.SetCreatedTime(now)
				event.SetDtStampTime(now)
				event.SetModifiedAt(now)

				if vessel.Type == "inport" {
					today := time.Now()
					start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
					end := start.Add(24 * time.Hour)

					event.SetAllDayStartAt(start)
					event.SetAllDayEndAt(end)

					event.SetSummary(fmt.Sprintf("Vessel In Port: %s", vessel.Name))
				} else {
					originalArrival, err := time.Parse("02/01/2006 15:04", vessel.Date+" "+vessel.Time)
					if err != nil {
						logger.Logger.Errorf("Error parsing vessel time for %s: %v", vessel.Name, err)
						continue
					}
					today := now
					start := time.Date(today.Year(), today.Month(), today.Day(),
						originalArrival.Hour(), originalArrival.Minute(), 0, 0, today.Location())
					end := start.Add(15 * time.Minute)

					event.SetStartAt(start)
					event.SetEndAt(end)
					event.SetSummary(fmt.Sprintf("Vessel In Port: %s", vessel.Name))
				}

				vesselLocation := vessel.LocationName
				if vesselLocation == "" {
					vesselLocation = strings.TrimSpace(vessel.LocationFrom + " " + vessel.LocationTo)
				}
				event.SetLocation(vesselLocation)
				event.SetDescription(fmt.Sprintf("Arrived: %s | Voyage: %s",
					vessel.Date+" "+vessel.Time, vessel.VoyageNo))
			}
		}
	}

	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}

// Healthz returns 200 OK if dependencies are healthy, 503 otherwise.
func (h *APIHandler) Healthz(c *fiber.Ctx) error {
	if err := h.svc.HealthCheck(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "fail", "error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}
