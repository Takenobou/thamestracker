package api

import (
	"strings"

	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gofiber/fiber/v2"
)

// FilterUniqueLifts filters out common bridge lifts, returning only unique ones.
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

// FilterShips applies various query filters on a list of ships.
func FilterShips(ships []models.Ship, c *fiber.Ctx) []models.Ship {
	nameFilter := strings.ToLower(c.Query("name", ""))
	locationFilter := strings.ToLower(c.Query("location", ""))
	nationalityFilter := strings.ToLower(c.Query("nationality", ""))
	afterFilter := c.Query("after", "")
	beforeFilter := c.Query("before", "")

	var result []models.Ship
	for _, ship := range ships {
		if nameFilter != "" && !strings.Contains(strings.ToLower(ship.Name), nameFilter) {
			continue
		}
		if locationFilter != "" &&
			!strings.Contains(strings.ToLower(ship.LocationFrom), locationFilter) &&
			!strings.Contains(strings.ToLower(ship.LocationTo), locationFilter) &&
			!strings.Contains(strings.ToLower(ship.LocationName), locationFilter) {
			continue
		}
		if nationalityFilter != "" && !strings.Contains(strings.ToLower(ship.Nationality), nationalityFilter) {
			continue
		}
		combinedDateTime, err := time.Parse("02/01/2006 15:04", ship.Date+" "+ship.Time)
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
		result = append(result, ship)
	}
	return result
}
