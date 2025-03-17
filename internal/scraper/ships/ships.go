package ships

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
)

// API response structure
type apiResponse struct {
	InPort     []shipData `json:"inport"`
	Arrivals   []shipData `json:"arrivals"`
	Departures []shipData `json:"departures"`
	Forecast   []shipData `json:"forecast"`
}

// shipData represents a generic ship record in the API response
type shipData struct {
	LocationFrom string `json:"location_from,omitempty"`
	LocationTo   string `json:"location_to,omitempty"`
	LocationName string `json:"location_name,omitempty"`
	VesselName   string `json:"vessel_name"`
	Visit        string `json:"visit"`
	Nationality  string `json:"nationality,omitempty"`
	LastRepDT    string `json:"last_rep_dt,omitempty"`
	FirstRepDT   string `json:"first_rep_dt,omitempty"` // Used in departures
	ETADate      string `json:"etad_dt,omitempty"`      // Used in forecast
}

// ScrapeShips fetches ship data based on the type (arrivals, departures, inport, forecast)
func ScrapeShips(shipType string) ([]models.Ship, error) {
	apiURL := config.AppConfig.URLs.PortOfLondon
	if apiURL == "" {
		logger.Logger.Errorf("Port of London API URL is missing from config.toml")
		return nil, fmt.Errorf("missing api url")
	}
	logger.Logger.Infof("Fetching ships from API, url: %s, shipType: %s", apiURL, shipType)
	resp, err := httpclient.DefaultClient.Get(apiURL)
	if err != nil {
		logger.Logger.Errorf("Error fetching ships: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Logger.Errorf("Error decoding API response: %v", err)
		return nil, err
	}

	var ships []models.Ship

	// Helper function to process ship data from a given category.
	processShips := func(shipList []shipData, category string) {
		for _, item := range shipList {
			var timestamp string
			switch category {
			case "departures":
				timestamp = item.FirstRepDT
			case "forecast":
				timestamp = item.ETADate
			default:
				timestamp = item.LastRepDT
			}

			if timestamp == "" {
				logger.Logger.Warnf("Missing timestamp for vessel: %s", item.VesselName)
				continue
			}

			parsedTime, err := time.Parse("2006-01-02 15:04:05.000", timestamp)
			if err != nil {
				logger.Logger.Errorf("Error parsing time for vessel %s, timestamp %s: %v", item.VesselName, timestamp, err)
				continue
			}

			ships = append(ships, models.Ship{
				Time:         parsedTime.Format("15:04"),      // HH:MM format
				Date:         parsedTime.Format("02/01/2006"), // DD/MM/YYYY format
				LocationFrom: item.LocationFrom,
				LocationTo:   item.LocationTo,
				LocationName: item.LocationName,
				Name:         item.VesselName,
				Nationality:  item.Nationality,
				VoyageNo:     item.Visit,
				Type:         category, // Set the type based on the category processed
			})
		}
	}

	// Handle "all" by processing each category.
	if shipType == "all" {
		processShips(result.InPort, "inport")
		processShips(result.Arrivals, "arrivals")
		processShips(result.Departures, "departures")
		processShips(result.Forecast, "forecast")
	} else {
		var shipList []shipData
		switch shipType {
		case "inport":
			shipList = result.InPort
		case "arrivals":
			shipList = result.Arrivals
		case "departures":
			shipList = result.Departures
		case "forecast":
			shipList = result.Forecast
		default:
			return nil, fmt.Errorf("invalid shipType: %s", shipType)
		}
		processShips(shipList, shipType)
	}

	logger.Logger.Infof("Retrieved ships from API, count: %d, shipType: %s", len(ships), shipType)
	return ships, nil
}
