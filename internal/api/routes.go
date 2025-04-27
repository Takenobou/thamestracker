package api

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// SetupRoutes initialises routes using the provided API handler.
func SetupRoutes(app *fiber.App, handler *APIHandler) {
	app.Get("/bridge-lifts", handler.GetBridgeLifts)
	app.Get("/vessels", handler.GetVessels)
	app.Get("/calendar.ics", handler.CalendarHandler)
	app.Get("/healthz", handler.Healthz)
	// Prometheus metrics endpoint
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
