package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// QueryOptions holds common query parameters used across endpoints.
type QueryOptions struct {
	Category    string // "bridge", "inport", "arrivals", etc.
	Location    string
	Unique      bool   // true if "unique" is set to "true"
	Name        string // filter by vessel or bridge name
	Nationality string // filter vessels by nationality
	After       string // after date/time filter
	Before      string // before date/time filter
}

// ParseQueryOptions parses common query parameters from the Fiber context.
func ParseQueryOptions(c *fiber.Ctx, defaultCategory string) QueryOptions {
	category := strings.ToLower(c.Query("category", ""))
	if category == "" {
		category = strings.ToLower(c.Query("type", defaultCategory))
	}
	return QueryOptions{
		Category:    category,
		Location:    strings.ToLower(c.Query("location", "")),
		Unique:      strings.EqualFold(c.Query("unique", "false"), "true"),
		Name:        strings.ToLower(c.Query("name", "")),
		Nationality: strings.ToLower(c.Query("nationality", "")),
		After:       c.Query("after", ""),
		Before:      c.Query("before", ""),
	}
}
