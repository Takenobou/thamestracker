package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// QueryOptions holds common query parameters used across endpoints.
type QueryOptions struct {
	EventType string // "all", "bridge", "ship"
	Location  string
	Unique    bool   // true if "unique" is set to "true"
	ShipType  string // for ship endpoints: "all", "inport", "arrivals", etc.
}

// ParseQueryOptions parses common query parameters from the Fiber context.
func ParseQueryOptions(c *fiber.Ctx) QueryOptions {
	opts := QueryOptions{
		EventType: strings.ToLower(c.Query("eventType", "all")),
		Location:  strings.ToLower(c.Query("location", "")),
		ShipType:  strings.ToLower(c.Query("type", "all")),
	}
	opts.Unique = c.Query("unique", "false") == "true"
	return opts
}
