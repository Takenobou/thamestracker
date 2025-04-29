package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	calendar "github.com/Takenobou/thamestracker/internal/calendar"
	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/metrics"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	ics "github.com/arran4/golang-ical"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
)

// BridgeSvc defines interface for bridge lift methods.
type BridgeSvc interface {
	GetBridgeLifts() ([]models.BridgeLift, error)
}

// VesselSvc defines interface for vessel methods.
type VesselSvc interface {
	GetVessels(vesselType string) ([]models.Vessel, error)
	GetFilteredVessels(vesselType, location string) ([]models.Vessel, error)
}

// HealthSvc defines interface for health check.
type HealthSvc interface {
	HealthCheck(ctx context.Context) error
}

// LocationSvc defines interface for location stats.
type LocationSvc interface {
	ListLocations() ([]service.LocationStats, error)
}

// ServiceInterface combines all service interfaces (for backwards compatibility).
type ServiceInterface interface {
	BridgeSvc
	VesselSvc
	HealthSvc
	LocationSvc
}

// APIHandler holds separate service interfaces.
type APIHandler struct {
	bridge   BridgeSvc
	vessel   VesselSvc
	health   HealthSvc
	location LocationSvc
}

// NewAPIHandler creates APIHandler from a combined service.
func NewAPIHandler(svc ServiceInterface) *APIHandler {
	return &APIHandler{bridge: svc, vessel: svc, health: svc, location: svc}
}

func (h *APIHandler) GetBridgeLifts(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c)
	lifts, err := h.bridge.GetBridgeLifts()
	if err != nil {
		// circuit breaker open -> 503 with Retry-After
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
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
	// validate vessel type early for client errors
	validTypes := map[string]bool{"all": true, "inport": true, "arrivals": true, "departures": true, "forecast": true}
	if !validTypes[opts.VesselType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("invalid type: %s", opts.VesselType)})
	}
	// parse and validate after/before filters
	if opts.After != "" {
		if _, err := time.Parse(time.RFC3339, opts.After); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid after parameter"})
		}
	}
	if opts.Before != "" {
		if _, err := time.Parse(time.RFC3339, opts.Before); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid before parameter"})
		}
	}
	// fetch and cache filtered vessels by type and location
	vessels, err := h.vessel.GetFilteredVessels(opts.VesselType, opts.Location)
	if err != nil {
		// circuit breaker open -> 503
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		// map invalid type errors to 400
		if strings.HasPrefix(err.Error(), "invalid vesselType") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		logger.Logger.Errorf("Error fetching vessel data: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve vessel data"})
	}
	// Apply JSON endpoint filters: name, nationality, after, before
	vessels = utils.FilterVessels(vessels, c)
	if opts.Unique {
		vessels = utils.FilterUniqueVessels(vessels)
	}
	return c.JSON(vessels)
}

// BridgeCalendarHandler returns iCalendar feed with only bridge lift events.
func (h *APIHandler) BridgeCalendarHandler(c *fiber.Ctx) error {
	// Load Europe/London timezone once
	loc, errLoc := time.LoadLocation("Europe/London")
	if errLoc != nil {
		logger.Logger.Errorf("Error loading timezone Europe/London: %v", errLoc)
		loc = time.UTC
	}
	// Get calendar options (e.g., unique, name filters)
	opts := ParseQueryOptions(c)
	// parse date-range filters
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")
	var fromTime, toTime, toEnd time.Time
	if fromStr != "" {
		ft, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid from parameter"})
		}
		fromTime = ft
	}
	if toStr != "" {
		tt, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid to parameter"})
		}
		toTime = tt
		toEnd = toTime.Add(24 * time.Hour)
	}
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	cal.SetRefreshInterval("PT1H")
	// Add London timezone to calendar
	cal.SetXWRTimezone("Europe/London")
	cal.AddVTimezone(ics.NewTimezone("Europe/London"))
	// fetch bridge lifts
	lifts, err := h.bridge.GetBridgeLifts()
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
	} else {
		if opts.Unique {
			lifts = utils.FilterUniqueLifts(lifts, 4)
		}
		lifts = utils.FilterBridgeLiftsByName(lifts, opts.Name)
		for _, lift := range lifts {
			start, err := time.ParseInLocation("2006-01-02 15:04", lift.Date+" "+lift.Time, loc)
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
	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}

// VesselsCalendarHandler returns iCalendar feed with only vessel events.
func (h *APIHandler) VesselsCalendarHandler(c *fiber.Ctx) error {
	// Load Europe/London timezone once
	loc, errLoc := time.LoadLocation("Europe/London")
	if errLoc != nil {
		logger.Logger.Errorf("Error loading timezone Europe/London: %v", errLoc)
		loc = time.UTC
	}
	// Get calendar options (e.g., unique filter)
	opts := ParseQueryOptions(c)
	// parse date-range filters
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")
	var fromTime, toTime, toEnd time.Time
	if fromStr != "" {
		ft, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid from parameter"})
		}
		fromTime = ft
	}
	if toStr != "" {
		tt, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid to parameter"})
		}
		toTime = tt
		toEnd = toTime.Add(24 * time.Hour)
	}
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	cal.SetRefreshInterval("PT1H")
	// Add London timezone to calendar
	cal.SetXWRTimezone("Europe/London")
	cal.AddVTimezone(ics.NewTimezone("Europe/London"))
	now := time.Now()

	vessels, err := h.vessel.GetVessels(opts.VesselType)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
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
				orig, err := time.ParseInLocation("02/01/2006 15:04", vessel.Date+" "+vessel.Time, loc)
				if err != nil {
					logger.Logger.Errorf("Error parsing vessel time for %s: %v", vessel.Name, err)
					continue
				}
				start = orig
				end = orig.Add(15 * time.Minute)
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
	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}

// Healthz returns 200 OK if dependencies are healthy, 503 otherwise.
func (h *APIHandler) Healthz(c *fiber.Ctx) error {
	if err := h.health.HealthCheck(c.UserContext()); err != nil {
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
	stats, err := h.location.ListLocations()
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
