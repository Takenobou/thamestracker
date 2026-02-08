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
	GetBridgeLifts() ([]models.Event, error)
}

// VesselSvc defines interface for vessel methods.
type VesselSvc interface {
	GetVessels(vesselType string) ([]models.Event, error)
	GetFilteredVessels(vesselType, location string) ([]models.Event, error)
}

// HealthSvc defines interface for health check.
type HealthSvc interface {
	HealthCheck(ctx context.Context) error
}

// ReadinessSvc defines interface for readiness check.
type ReadinessSvc interface {
	ReadyCheck(ctx context.Context) error
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
	ReadinessSvc
	LocationSvc
}

// APIHandler holds separate service interfaces.
type APIHandler struct {
	bridge    BridgeSvc
	vessel    VesselSvc
	health    HealthSvc
	readiness ReadinessSvc
	location  LocationSvc
}

// NewAPIHandler creates APIHandler from a combined service.
func NewAPIHandler(svc ServiceInterface) *APIHandler {
	return &APIHandler{bridge: svc, vessel: svc, health: svc, readiness: svc, location: svc}
}

func (h *APIHandler) GetBridgeLifts(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c, "bridge")
	if err := validateBridgeQueryOptions(opts); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if opts.Category == "all" {
		opts.Category = "bridge"
	}

	events, err := h.bridge.GetBridgeLifts()
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
	}
	filtered := utils.FilterEvents(events, utils.FilterOptions{
		Name:                   opts.Name,
		Category:               opts.Category,
		After:                  opts.After,
		Before:                 opts.Before,
		Unique:                 opts.Unique,
		Location:               opts.Location,
		BridgeFilterPercentile: config.AppConfig.BridgeFilterPercentile,
		BridgeFilterMaxCount:   config.AppConfig.BridgeFilterMaxCount,
	})
	return c.JSON(filtered)
}

func (h *APIHandler) GetVessels(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c, "all")
	if err := validateVesselQueryOptions(opts); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	events, err := h.vessel.GetVessels(opts.Category)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		if strings.HasPrefix(err.Error(), "invalid vesselType") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		logger.Logger.Errorf("Error fetching vessel data: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve vessel data"})
	}
	filtered := utils.FilterEvents(events, utils.FilterOptions{
		Name:        opts.Name,
		Category:    opts.Category,
		Nationality: opts.Nationality,
		After:       opts.After,
		Before:      opts.Before,
		Unique:      opts.Unique,
		Location:    opts.Location,
	})
	return c.JSON(filtered)
}

// BridgeCalendarHandler returns iCalendar feed with only bridge lift events.
func (h *APIHandler) BridgeCalendarHandler(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c, "bridge")
	if err := validateBridgeQueryOptions(opts); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if opts.Category == "all" {
		opts.Category = "bridge"
	}

	events, err := h.bridge.GetBridgeLifts()
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		logger.Logger.Errorf("Error fetching bridge lifts: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
	}
	filtered := utils.FilterEvents(events, utils.FilterOptions{
		Name:                   opts.Name,
		Category:               opts.Category,
		After:                  opts.After,
		Before:                 opts.Before,
		Unique:                 opts.Unique,
		Location:               opts.Location,
		BridgeFilterPercentile: config.AppConfig.BridgeFilterPercentile,
		BridgeFilterMaxCount:   config.AppConfig.BridgeFilterMaxCount,
	})
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	cal.SetRefreshInterval("PT1H")
	cal.SetXWRTimezone("Europe/London")
	cal.AddVTimezone(ics.NewTimezone("Europe/London"))
	for _, e := range filtered {
		calendar.BuildEvent(cal, e)
	}
	c.Set("Content-Type", "text/calendar")
	return c.SendString(cal.Serialize())
}

// VesselsCalendarHandler returns iCalendar feed with only vessel events.
func (h *APIHandler) VesselsCalendarHandler(c *fiber.Ctx) error {
	opts := ParseQueryOptions(c, "all")
	if err := validateVesselQueryOptions(opts); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	events, err := h.vessel.GetVessels(opts.Category)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			c.Set("Retry-After", strconv.Itoa(config.AppConfig.CircuitBreaker.CoolOffSeconds))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service temporarily unavailable"})
		}
		logger.Logger.Errorf("Error fetching vessel data: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve vessel data"})
	}
	filtered := utils.FilterEvents(events, utils.FilterOptions{
		Name:        opts.Name,
		Category:    opts.Category,
		Nationality: opts.Nationality,
		After:       opts.After,
		Before:      opts.Before,
		Unique:      opts.Unique,
		Location:    opts.Location,
	})
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//ThamesTracker//EN")
	cal.SetRefreshInterval("PT1H")
	cal.SetXWRTimezone("Europe/London")
	cal.AddVTimezone(ics.NewTimezone("Europe/London"))
	for _, e := range filtered {
		calendar.BuildEvent(cal, e)
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

// Readyz returns 200 OK if dependencies are ready, 503 otherwise.
func (h *APIHandler) Readyz(c *fiber.Ctx) error {
	if err := h.readiness.ReadyCheck(c.UserContext()); err != nil {
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

func validateVesselQueryOptions(opts QueryOptions) error {
	validTypes := map[string]bool{"all": true, "inport": true, "arrivals": true, "departures": true, "forecast": true}
	if !validTypes[opts.Category] {
		return fmt.Errorf("invalid type: %s", opts.Category)
	}
	return validateTimeRange(opts.After, opts.Before)
}

func validateBridgeQueryOptions(opts QueryOptions) error {
	if opts.Category != "" && opts.Category != "bridge" && opts.Category != "all" {
		return fmt.Errorf("invalid type: %s", opts.Category)
	}
	return validateTimeRange(opts.After, opts.Before)
}

func validateTimeRange(after string, before string) error {
	var afterTS time.Time
	var beforeTS time.Time
	var err error

	if after != "" {
		afterTS, err = time.Parse(time.RFC3339, after)
		if err != nil {
			return fmt.Errorf("invalid after parameter")
		}
	}
	if before != "" {
		beforeTS, err = time.Parse(time.RFC3339, before)
		if err != nil {
			return fmt.Errorf("invalid before parameter")
		}
	}
	if !afterTS.IsZero() && !beforeTS.IsZero() && afterTS.After(beforeTS) {
		return fmt.Errorf("after must be before or equal to before")
	}
	return nil
}
