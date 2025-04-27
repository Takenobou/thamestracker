package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// QueryOptions holds common query parameters used across endpoints.
type QueryOptions struct {
	EventType   string // "all", "bridge", "vessel"
	Location    string
	Unique      bool   // true if "unique" is set to "true"
	VesselType  string // for vessel endpoints: "all", "inport", "arrivals", etc.
	Name        string // filter by vessel or bridge name
	Nationality string // filter vessels by nationality
	After       string // after date/time filter
	Before      string // before date/time filter
}

// ParseQueryOptions parses common query parameters from the Fiber context.
func ParseQueryOptions(c *fiber.Ctx) QueryOptions {
	opts := QueryOptions{
		EventType:  strings.ToLower(c.Query("eventType", "all")),
		Location:   strings.ToLower(c.Query("location", "")),
		VesselType: strings.ToLower(c.Query("type", "all")),
	}
	opts.Unique = c.Query("unique", "false") == "true"
	opts.Name = strings.ToLower(c.Query("name", ""))
	opts.Nationality = strings.ToLower(c.Query("nationality", ""))
	opts.After = c.Query("after", "")
	opts.Before = c.Query("before", "")
	return opts
}
