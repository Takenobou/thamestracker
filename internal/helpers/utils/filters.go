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
