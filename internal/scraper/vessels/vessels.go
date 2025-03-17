package vessels

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
)

// apiResponse represents the API response structure.
type apiResponse struct {
	InPort     []vesselData `json:"inport"`
	Arrivals   []vesselData `json:"arrivals"`
	Departures []vesselData `json:"departures"`
	Forecast   []vesselData `json:"forecast"`
}

// vesselData represents a generic vessel record in the API response.
type vesselData struct {
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

// ScrapeVessels fetches vessel data based on the type (arrivals, departures, inport, forecast).
func ScrapeVessels(vesselType string) ([]models.Vessel, error) {
	apiURL := config.AppConfig.URLs.PortOfLondon
	if apiURL == "" {
		logger.Logger.Errorf("Port of London API URL is missing from config.toml")
		return nil, fmt.Errorf("missing api url")
	}
	logger.Logger.Infof("Fetching vessels from API, url: %s, vesselType: %s", apiURL, vesselType)
	resp, err := httpclient.DefaultClient.Get(apiURL)
	if err != nil {
		logger.Logger.Errorf("Error fetching vessels: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Logger.Errorf("Error decoding API response: %v", err)
		return nil, err
	}

	var vessels []models.Vessel

	// Helper function to process vessel data from a given category.
	processVessels := func(vesselList []vesselData, category string) {
		for _, item := range vesselList {
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

			vessels = append(vessels, models.Vessel{
				Time:         parsedTime.Format("15:04"),
				Date:         parsedTime.Format("02/01/2006"),
				LocationFrom: item.LocationFrom,
				LocationTo:   item.LocationTo,
				LocationName: item.LocationName,
				Name:         item.VesselName,
				Nationality:  item.Nationality,
				VoyageNo:     item.Visit,
				Type:         category,
			})
		}
	}

	// Handle "all" by processing each category.
	if vesselType == "all" {
		processVessels(result.InPort, "inport")
		processVessels(result.Arrivals, "arrivals")
		processVessels(result.Departures, "departures")
		processVessels(result.Forecast, "forecast")
	} else {
		var vesselList []vesselData
		switch vesselType {
		case "inport":
			vesselList = result.InPort
		case "arrivals":
			vesselList = result.Arrivals
		case "departures":
			vesselList = result.Departures
		case "forecast":
			vesselList = result.Forecast
		default:
			return nil, fmt.Errorf("invalid vesselType: %s", vesselType)
		}
		processVessels(vesselList, vesselType)
	}

	logger.Logger.Infof("Retrieved vessels from API, count: %d, vesselType: %s", len(vessels), vesselType)
	return vessels, nil
}
