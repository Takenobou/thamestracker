package api

import (
	"log"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	shipScraper "github.com/Takenobou/thamestracker/internal/scraper/ships"
	"github.com/Takenobou/thamestracker/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// GetBridgeLifts returns all scheduled bridge lifts (or unique ones if ?unique=true is set)
func GetBridgeLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	log.Println("Checking Redis cache for bridge lifts...")
	err := storage.GetCache("bridge_lifts", &lifts)
	if err == nil {
		log.Println("Returning cached bridge lifts ✅")
	} else {
		log.Println("Cache miss ❌, scraping fresh bridge lift data...")
		lifts, err = bridgeScraper.ScrapeBridgeLifts()
		if err != nil {
			log.Println("Error fetching bridge lifts:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
		}
		storage.SetCache("bridge_lifts", lifts, 15*time.Minute)
	}

	// Filter unique lifts if query parameter is set
	unique := c.Query("unique", "false")
	if unique == "true" {
		lifts = FilterUniqueLifts(lifts)
	}

	return c.JSON(lifts)
}

// FilterUniqueLifts filters out common lifts, returning only the unique ones.
func FilterUniqueLifts(lifts []models.BridgeLift) []models.BridgeLift {
	vesselCount := make(map[string]int)
	for _, lift := range lifts {
		vesselCount[lift.Vessel]++
	}

	var uniqueLifts []models.BridgeLift
	for _, lift := range lifts {
		if vesselCount[lift.Vessel] <= 2 { // Adjust the threshold if needed
			uniqueLifts = append(uniqueLifts, lift)
		}
	}

	return uniqueLifts
}

// GetShips handles all ship data queries by type using a query parameter.
// Use ?type=all (default), ?type=inport, ?type=arrivals, ?type=departures, or ?type=forecast.
func GetShips(c *fiber.Ctx) error {
	shipType := c.Query("type", "all")
	var cacheKey string
	if shipType == "all" {
		cacheKey = "all_ships"
	} else {
		cacheKey = "ships_" + strings.ToLower(shipType)
	}
	return GetShipData(shipType, cacheKey, c)
}

// GetShipData fetches ships from the API (or Redis cache) and applies filtering based on query parameters.
func GetShipData(shipType, cacheKey string, c *fiber.Ctx) error {
	var ships []models.Ship

	// Check Redis cache first
	err := storage.GetCache(cacheKey, &ships)
	if err == nil {
		log.Println("Returning cached ship data ✅")
	} else {
		// Cache miss - scrape fresh data
		log.Println("Cache miss ❌, scraping fresh data...")
		ships, err = shipScraper.ScrapeShips(shipType)
		if err != nil {
			log.Println("Error fetching ship data:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve ship data"})
		}
		storage.SetCache(cacheKey, ships, 30*time.Minute)
	}

	// **Apply Filtering** based on additional query parameters.
	filteredShips := FilterShips(ships, c)
	return c.JSON(filteredShips)
}

// FilterShips applies various query filters (name, location, nationality, time range) on the ship list.
func FilterShips(ships []models.Ship, c *fiber.Ctx) []models.Ship {
	nameFilter := strings.ToLower(c.Query("name", ""))
	locationFilter := strings.ToLower(c.Query("location", ""))
	nationalityFilter := strings.ToLower(c.Query("nationality", ""))
	afterFilter := c.Query("after", "")
	beforeFilter := c.Query("before", "")

	var result []models.Ship
	for _, ship := range ships {
		// Apply name filter
		if nameFilter != "" && !strings.Contains(strings.ToLower(ship.Name), nameFilter) {
			continue
		}

		// Apply location filter (checks multiple fields)
		if locationFilter != "" &&
			!strings.Contains(strings.ToLower(ship.LocationFrom), locationFilter) &&
			!strings.Contains(strings.ToLower(ship.LocationTo), locationFilter) &&
			!strings.Contains(strings.ToLower(ship.LocationName), locationFilter) {
			continue
		}

		// Apply nationality filter
		if nationalityFilter != "" && !strings.Contains(strings.ToLower(ship.Nationality), nationalityFilter) {
			continue
		}

		// Combine Date and Time into a single time.Time object for filtering
		combinedDateTime, err := time.Parse("02/01/2006 15:04", ship.Date+" "+ship.Time)
		if err != nil {
			continue // Skip if parsing fails
		}

		// Apply "after" filter
		if afterFilter != "" {
			parsedAfter, err := time.Parse(time.RFC3339, afterFilter)
			if err == nil && combinedDateTime.Before(parsedAfter) {
				continue
			}
		}

		// Apply "before" filter
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
