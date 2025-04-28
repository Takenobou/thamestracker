package api

import (
	"strconv"
	"strings"
	"time"

	calendar "github.com/Takenobou/thamestracker/internal/calendar"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/metrics"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	ics "github.com/arran4/golang-ical"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type ServiceInterface interface {
	GetBridgeLifts() ([]models.BridgeLift, error)
	GetVessels(vesselType string) ([]models.Vessel, error)
	HealthCheck() error
	ListLocations() ([]service.LocationStats, error)
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
	// Name filtering if specified.
	lifts = utils.FilterBridgeLiftsByName(lifts, opts.Name)
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

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	// VCALENDAR refresh hint
	cal.SetRefreshInterval("PT1H")
	now := time.Now()

	// parse optional date-range filters
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")
	var fromTime, toTime, toEnd time.Time
	if fromStr != "" {
		if ft, err := time.Parse("2006-01-02", fromStr); err == nil {
			fromTime = ft
		}
	}
	if toStr != "" {
		if tt, err := time.Parse("2006-01-02", toStr); err == nil {
			toTime = tt
			toEnd = toTime.Add(24 * time.Hour)
		}
	}

	// Bridge events (only when explicitly requested)
	if opts.EventType == "bridge" {
		lifts, err := h.svc.GetBridgeLifts()
		if err != nil {
			logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		} else {
			if opts.Unique {
				lifts = utils.FilterUniqueLifts(lifts, 4)
			}
			lifts = utils.FilterBridgeLiftsByName(lifts, opts.Name)
			for _, lift := range lifts {
				start, err := time.Parse("2006-01-02 15:04", lift.Date+" "+lift.Time)
				if err != nil {
					logger.Logger.Errorf("Error parsing bridge lift time for vessel %s: %v", lift.Vessel, err)
					continue
				}
				if fromStr != "" && start.Before(fromTime) {
					continue
				}
				if toStr != "" && start.After(toEnd) {
					continue
				}
				calendar.BuildBridgeEvent(cal, lift, start)
			}
		}
	}

	// Vessel events (inport/arrivals/departures/forecast as well as all)
	if opts.EventType != "bridge" {
		vessels, err := h.svc.GetVessels(opts.VesselType)
		if err != nil {
			logger.Logger.Errorf("Error fetching vessel data: %v", err)
		} else {
			vessels = utils.FilterVessels(vessels, c)
			if opts.Unique {
				vessels = utils.FilterUniqueVessels(vessels)
			}
			for _, vessel := range vessels {
				var start, end time.Time
				if vessel.Type == "inport" {
					today := now
					start = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
					end = start.Add(24 * time.Hour)
				} else {
					orig, err := time.Parse("02/01/2006 15:04", vessel.Date+" "+vessel.Time)
					if err != nil {
						logger.Logger.Errorf("Error parsing vessel time for %s: %v", vessel.Name, err)
						continue
					}
					today := now
					start = time.Date(today.Year(), today.Month(), today.Day(), orig.Hour(), orig.Minute(), 0, 0, today.Location())
					end = start.Add(15 * time.Minute)
				}
				if fromStr != "" && start.Before(fromTime) {
					continue
				}
				if toStr != "" && start.After(toEnd) {
					continue
				}
				calendar.BuildVesselEvent(cal, vessel, start, end)
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

// GetLocations handles GET /locations endpoint.
func (h *APIHandler) GetLocations(c *fiber.Ctx) error {
	// metrics
	timer := prometheus.NewTimer(metrics.LocationsRequestDuration)
	defer timer.ObserveDuration()
	metrics.LocationsRequests.Inc()

	// parse query params
	minTotal, err := strconv.Atoi(c.Query("minTotal", "0"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid minTotal"})
	}
	q := strings.ToLower(c.Query("q", ""))
	// get aggregated stats
	stats, err := h.svc.ListLocations()
	if err != nil {
		logger.Logger.Errorf("Error listing locations: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve location data"})
	}
	// filter and return
	var out []service.LocationStats
	for _, s := range stats {
		if s.Total < minTotal {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(s.Name), q) {
			continue
		}
		out = append(out, s)
	}
	// log at most one structured line
	logger.Logger.Infof("path=/locations hits=%d", len(out))
	return c.JSON(out)
}
