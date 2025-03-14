package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Takenobou/thamestracker/config"
)

// Ship represents a vessel in port
type Ship struct {
	Time     string `json:"time"`
	Date     string `json:"date"`
	Location string `json:"location"`
	Name     string `json:"name"`
	VoyageNo string `json:"voyage_number"`
}

// API response structure
type apiResponse struct {
	InPort []struct {
		LocationName string `json:"location_name"`
		VesselName   string `json:"vessel_name"`
		Visit        string `json:"visit"`
		LastRepDT    string `json:"last_rep_dt"` // Timestamp for arrival
	} `json:"inport"`
}

// ScrapeShips fetches in-port ships from the API URL in config
func ScrapeShips() ([]Ship, error) {
	apiURL := config.AppConfig.URLs.PortOfLondon
	if apiURL == "" {
		log.Println("❌ Error: Port of London API URL is missing from config.toml")
		return nil, fmt.Errorf("missing api url")
	}
	log.Println("🔹 Fetching ships from API:", apiURL)

	// Make the HTTP request
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Println("❌ Error fetching ships:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Decode JSON response
	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("❌ Error decoding API response:", err)
		return nil, err
	}

	// Convert API response to Ship struct
	var ships []Ship
	for _, item := range result.InPort {
		// Convert timestamp to readable date/time
		parsedTime, _ := time.Parse("2006-01-02 15:04:05.000", item.LastRepDT)

		ships = append(ships, Ship{
			Time:     parsedTime.Format("15:04"), // HH:MM format
			Date:     parsedTime.Format("02/01/2006"),
			Location: item.LocationName,
			Name:     item.VesselName,
			VoyageNo: item.Visit,
		})
	}

	log.Printf("✅ Retrieved %d ships from API\n", len(ships))
	return ships, nil
}
