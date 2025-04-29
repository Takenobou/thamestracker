package vessels

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/helpers/utils"
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
		logger.Logger.Errorf("Port of London API URL is missing: set PORT_OF_LONDON environment variable")
		return nil, fmt.Errorf("missing api url")
	}
	logger.Logger.Infof("Fetching vessels from API, url: %s, vesselType: %s", apiURL, vesselType)

	// Retry GET with exponential backoff
	var resp *http.Response
	err := utils.Retry(3, 500*time.Millisecond, func() error {
		r, e := httpclient.DefaultClient.Get(apiURL)
		if e != nil {
			logger.Logger.Warnf("Fetch failed: %v", e)
			return e
		}
		if r.StatusCode >= http.StatusInternalServerError {
			r.Body.Close()
			e = fmt.Errorf("server error %d", r.StatusCode)
			logger.Logger.Warnf("Fetch failed: %v", e)
			return e
		}
		resp = r
		return nil
	})
	if err != nil {
		logger.Logger.Errorf("Error fetching vessels after retries: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Logger.Errorf("Error decoding API response: %v", err)
		return nil, err
	}
	// summary log for fetched vessels counts
	logger.Logger.Infof("PLA vessels fetched: inport=%d arrivals=%d departures=%d forecast=%d url=%s",
		len(result.InPort), len(result.Arrivals), len(result.Departures), len(result.Forecast), apiURL)

	vessels := make([]models.Vessel, 0)

	// Helper function to process vessel data from a given category.
	processVessels := func(vesselList []vesselData, category string) {
		for _, item := range vesselList {
			if item.VesselName == "" {
				logger.Logger.Warnf("Missing vessel name in category %s, skipping", category)
				continue
			}
			// skip missing voyage number, except for forecasts
			if item.Visit == "" && category != "forecast" {
				logger.Logger.Warnf("Missing voyage number for vessel %s, skipping", item.VesselName)
				continue
			}

			var ts string
			switch category {
			case "departures":
				ts = item.FirstRepDT
			case "forecast":
				ts = item.ETADate
			default:
				ts = item.LastRepDT
			}
			if ts == "" {
				logger.Logger.Warnf("Missing timestamp for vessel %s, using now()", item.VesselName)
				ts = time.Now().Format("2006-01-02 15:04:05.000")
			}

			// extract date and time parts directly
			var dateStr, timeStr string
			if ts == "" {
				nowLocal := time.Now()
				dateStr = nowLocal.Format("02/01/2006")
				timeStr = nowLocal.Format("15:04")
			} else {
				// robustly parse timestamp with or without milliseconds
				layouts := []string{"2006-01-02 15:04:05.000", "2006-01-02 15:04:05"}
				var t time.Time
				var parseErr error
				for _, layout := range layouts {
					t, parseErr = time.Parse(layout, ts)
					if parseErr == nil {
						break
					}
				}
				if parseErr != nil {
					logger.Logger.Warnf("Timestamp parse failed for vessel %s: %v, using now()", item.VesselName, parseErr)
					t = time.Now()
				}
				dateStr = t.Format("02/01/2006")
				timeStr = t.Format("15:04")
			}

			vessels = append(vessels, models.Vessel{
				Time:         timeStr,
				Date:         dateStr,
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
