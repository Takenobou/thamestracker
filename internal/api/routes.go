package api

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// SetupRoutes initialises routes using the provided API handler.
func SetupRoutes(app *fiber.App, handler *APIHandler) {
	app.Get("/bridge-lifts", handler.GetBridgeLifts)
	app.Get("/vessels", handler.GetVessels)
	app.Get("/bridge-lifts/calendar.ics", handler.BridgeCalendarHandler)
	app.Get("/vessels/calendar.ics", handler.VesselsCalendarHandler)
	app.Get("/healthz", handler.Healthz)
	app.Get("/locations", handler.GetLocations)
	// Prometheus metrics endpoint (registered only when public)
	if config.AppConfig.MetricsPublic {
		app.Get("/metrics", func(c *fiber.Ctx) error {
			mfs, err := prometheus.DefaultGatherer.Gather()
			if err != nil {
				return c.Status(500).SendString(err.Error())
			}
			var buf bytes.Buffer
			for _, mf := range mfs {
				expfmt.MetricFamilyToText(&buf, mf)
			}
			c.Set("Content-Type", "text/plain; version=0.0.4")
			return c.SendString(buf.String())
		})
	}
	// Serve OpenAPI spec
	app.Get("/docs", func(c *fiber.Ctx) error {
		specPath := filepath.Join("docs", "openapi.json")
		data, err := os.ReadFile(specPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load OpenAPI spec"})
		}
		c.Set("Content-Type", "application/json")
		return c.Send(data)
	})
}
