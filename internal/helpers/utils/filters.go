package utils

import (
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gofiber/fiber/v2"
)

// FilterUniqueLifts filters out bridge lifts that appear more than a given threshold.
func FilterUniqueLifts(lifts []models.BridgeLift, threshold int) []models.BridgeLift {
	vesselCount := make(map[string]int)
	for _, lift := range lifts {
		vesselCount[lift.Vessel]++
	}

	var uniqueLifts []models.BridgeLift
	for _, lift := range lifts {
		if vesselCount[lift.Vessel] <= threshold {
			uniqueLifts = append(uniqueLifts, lift)
		}
	}
	return uniqueLifts
}

// FilterBridgeLiftsByName filters bridge lifts where Vessel contains the given lowercase substring.
func FilterBridgeLiftsByName(lifts []models.BridgeLift, name string) []models.BridgeLift {
	if name == "" {
		return lifts
	}
	var result []models.BridgeLift
	for _, lift := range lifts {
		if strings.Contains(strings.ToLower(lift.Vessel), name) {
			result = append(result, lift)
		}
	}
	return result
}

// FilterVessels applies various query filters on a list of vessels.
func FilterVessels(vessels []models.Vessel, c *fiber.Ctx) []models.Vessel {
	nameFilter := strings.ToLower(c.Query("name", ""))
	locationFilter := strings.ToLower(c.Query("location", ""))
	nationalityFilter := strings.ToLower(c.Query("nationality", ""))
	afterFilter := c.Query("after", "")
	beforeFilter := c.Query("before", "")

	var result []models.Vessel
	for _, vessel := range vessels {
		if nameFilter != "" && !strings.Contains(strings.ToLower(vessel.Name), nameFilter) {
			continue
		}
		if locationFilter != "" &&
			!strings.Contains(strings.ToLower(vessel.LocationFrom), locationFilter) &&
			!strings.Contains(strings.ToLower(vessel.LocationTo), locationFilter) &&
			!strings.Contains(strings.ToLower(vessel.LocationName), locationFilter) {
			continue
		}
		if nationalityFilter != "" && !strings.Contains(strings.ToLower(vessel.Nationality), nationalityFilter) {
			continue
		}
		combinedDateTime, err := time.Parse("02/01/2006 15:04", vessel.Date+" "+vessel.Time)
		if err != nil {
			continue
		}
		if afterFilter != "" {
			parsedAfter, err := time.Parse(time.RFC3339, afterFilter)
			if err == nil && combinedDateTime.Before(parsedAfter) {
				continue
			}
		}
		if beforeFilter != "" {
			parsedBefore, err := time.Parse(time.RFC3339, beforeFilter)
			if err == nil && combinedDateTime.After(parsedBefore) {
				continue
			}
		}
		result = append(result, vessel)
	}
	return result
}

// FilterUniqueVessels de-duplicates vessel entries by name when unique=true.
func FilterUniqueVessels(vessels []models.Vessel) []models.Vessel {
	seen := make(map[string]bool)
	var result []models.Vessel
	for _, v := range vessels {
		if !seen[v.Name] {
			seen[v.Name] = true
			result = append(result, v)
		}
	}
	return result
}

// FilterOptions defines generic filters for Event slices.
type FilterOptions struct {
	Name     string
	Category string
	After    string
	Before   string
	Unique   bool
	Location string
}

// FilterEvents applies generic filters to a slice of Event.
func FilterEvents(events []models.Event, opts FilterOptions) []models.Event {
	var filtered []models.Event
	name := strings.ToLower(opts.Name)
	category := strings.ToLower(opts.Category)
	location := strings.ToLower(opts.Location)
	var after, before time.Time
	var afterSet, beforeSet bool
	if opts.After != "" {
		t, err := time.Parse(time.RFC3339, opts.After)
		if err == nil {
			after = t
			afterSet = true
		}
	}
	if opts.Before != "" {
		t, err := time.Parse(time.RFC3339, opts.Before)
		if err == nil {
			before = t
			beforeSet = true
		}
	}
	for _, e := range events {
		if name != "" && !strings.Contains(strings.ToLower(e.VesselName), name) {
			continue
		}
		if category != "" && category != "all" && strings.ToLower(e.Category) != category {
			continue
		}
		if location != "" && !strings.Contains(strings.ToLower(e.Location), location) {
			continue
		}
		if afterSet && e.Timestamp.Before(after) {
			continue
		}
		if beforeSet && e.Timestamp.After(before) {
			continue
		}
		filtered = append(filtered, e)
	}
	if opts.Unique {
		filtered = filterUniqueEvents(filtered)
	}
	return filtered
}

// filterUniqueEvents de-duplicates events by VesselName and Category.
func filterUniqueEvents(events []models.Event) []models.Event {
	seen := make(map[string]bool)
	var result []models.Event
	for _, e := range events {
		key := e.Category + ":" + e.VesselName
		if !seen[key] {
			seen[key] = true
			result = append(result, e)
		}
	}
	return result
}
