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

// ScrapeVessels fetches vessel data as unified events based on the type (arrivals, departures, inport, forecast).
func ScrapeVessels(vesselType string) ([]models.Event, error) {
	apiURL := config.AppConfig.URLs.PortOfLondon
	if apiURL == "" {
		logger.Logger.Errorf("Port of London API URL is missing: set PORT_OF_LONDON environment variable")
		return nil, fmt.Errorf("missing api url")
	}
	logger.Logger.Infof("Fetching vessels from API, url: %s, vesselType: %s", apiURL, vesselType)

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
	logger.Logger.Infof("PLA vessels fetched: inport=%d arrivals=%d departures=%d forecast=%d url=%s",
		len(result.InPort), len(result.Arrivals), len(result.Departures), len(result.Forecast), apiURL)

	events := make([]models.Event, 0)

	parseTimestamp := func(raw string) time.Time {
		const srcLayout = "2006-01-02 15:04:05.000"
		london, _ := time.LoadLocation("Europe/London")
		if raw != "" {
			if t, err := time.ParseInLocation(srcLayout, raw, london); err == nil {
				return t
			}
		}
		return time.Now().In(london)
	}

	processVessels := func(vesselList []vesselData, category string) {
		for _, item := range vesselList {
			if item.VesselName == "" {
				logger.Logger.Warnf("Missing vessel name in category %s, skipping", category)
				continue
			}
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
			tParsed := parseTimestamp(ts)

			event := models.Event{
				Timestamp:  tParsed,
				VesselName: item.VesselName,
				Category:   category,
				VoyageNo:   item.Visit,
				From:       item.LocationFrom,
				To:         item.LocationTo,
				Location:   item.LocationName,
			}
			events = append(events, event)
		}
	}

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

	logger.Logger.Infof("Retrieved vessel events from API, count: %d, vesselType: %s", len(events), vesselType)
	return events, nil
}
