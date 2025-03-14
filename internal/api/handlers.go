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

// GetBridgeLifts - Returns all scheduled bridge lifts
func GetBridgeLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	log.Println("Checking Redis cache for bridge lifts...")
	err := storage.GetCache("bridge_lifts", &lifts)
	if err == nil {
		log.Println("Returning cached bridge lifts ✅")
		return c.JSON(lifts)
	}

	log.Println("Cache miss ❌, scraping fresh bridge lift data...")
	lifts, err = bridgeScraper.ScrapeBridgeLifts()
	if err != nil {
		log.Println("Error fetching bridge lifts:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
	}

	storage.SetCache("bridge_lifts", lifts, 15*time.Minute)
	return c.JSON(lifts)
}

// GetRareLifts - Returns only vessels that appear infrequently
func GetRareLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	// Try cache first
	err := storage.GetCache("bridge_lifts", &lifts)
	if err != nil {
		log.Println("Cache miss ❌, scraping fresh data...")
		lifts, err = bridgeScraper.ScrapeBridgeLifts()
		if err != nil {
			log.Println("Error fetching bridge lifts:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve data"})
		}
		// Store in cache
		storage.SetCache("bridge_lifts", lifts, 15*time.Minute)
	}

	// Count occurrences of each vessel
	vesselCount := make(map[string]int)
	for _, lift := range lifts {
		vesselCount[lift.Vessel]++
	}

	// Filter rare vessels (e.g., appears ≤ 2 times)
	var rareLifts []models.BridgeLift
	for _, lift := range lifts {
		if vesselCount[lift.Vessel] <= 2 {
			rareLifts = append(rareLifts, lift)
		}
	}

	return c.JSON(rareLifts)
}

// FilterShips applies query parameters to filter the ship list
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

		// Apply location filter (matches both `location_from` and `location_to`)
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

		// Convert Date & Time into a `time.Time` object
		combinedDateTime, err := time.Parse("02/01/2006 15:04", ship.Date+" "+ship.Time)
		if err != nil {
			continue // Skip this entry if parsing fails
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

		// If ship passes all filters, add it to results
		result = append(result, ship)
	}

	return result
}

// GetShipData fetches ships from the API and applies optional filters
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

	// **Apply Filtering**
	filteredShips := FilterShips(ships, c)

	return c.JSON(filteredShips)
}

func GetShips(c *fiber.Ctx) error {
	return GetShipData("inport", "ships_in_port", c)
}

// GetArrivals - Fetch currently arriving ships
func GetArrivals(c *fiber.Ctx) error {
	return GetShipData("arrivals", "ship_arrivals", c)
}

// GetDepartures - Fetch currently departing ships
func GetDepartures(c *fiber.Ctx) error {
	return GetShipData("departures", "ship_departures", c)
}

// GetForecast - Fetch upcoming ship movements
func GetForecast(c *fiber.Ctx) error {
	return GetShipData("forecast", "ship_forecast", c)
}
