package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Ensure logger is initialized to avoid nil pointer dereference.
	logger.InitLogger()
}

const mockAPIResponse = `{
	"inport": [
		{
			"location_name": "WOODS QUAY",
			"vessel_name": "SILVER STURGEON",
			"visit": "S7670",
			"last_rep_dt": "2025-01-25 20:33:47.150"
		}
	],
	"arrivals": [
		{
			"location_from": "MAPTM",
			"location_to": "LONDON GATEWAY1",
			"vessel_name": "SAN NICOLAS MAERSK",
			"visit": "S7795",
			"last_rep_dt": "2025-03-13 14:22:09.300"
		}
	],
	"departures": [
		{
			"location_from": "TILBURY DOCK",
			"location_to": "SESOE",
			"vessel_name": "FRISIAN SPRING",
			"visit": "F1785",
			"first_rep_dt": "2025-03-13 15:39:03.690"
		}
	],
	"forecast": [
		{
			"location_from": "NLVLI",
			"location_to": "FORDS JETTY",
			"vessel_name": "ADELINE",
			"visit": "A9999",
			"etad_dt": "2025-03-14 14:15:00.000"
		}
	]
}`

func TestScrapeVessels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockAPIResponse))
	}))
	defer server.Close()

	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = server.URL
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

	tests := []struct {
		name          string
		vesselType    string
		expectedCount int
		expected      []models.Event
	}{
		{
			name:          "Inport",
			vesselType:    "inport",
			expectedCount: 1,
			expected: []models.Event{{
				VesselName: "SILVER STURGEON",
				Category:   "inport",
				VoyageNo:   "S7670",
				Location:   "WOODS QUAY",
			}},
		},
		{
			name:          "Arrivals",
			vesselType:    "arrivals",
			expectedCount: 1,
			expected: []models.Event{{
				VesselName: "SAN NICOLAS MAERSK",
				Category:   "arrivals",
				VoyageNo:   "S7795",
				From:       "MAPTM",
				To:         "LONDON GATEWAY1",
			}},
		},
		{
			name:          "Departures",
			vesselType:    "departures",
			expectedCount: 1,
			expected: []models.Event{{
				VesselName: "FRISIAN SPRING",
				Category:   "departures",
				VoyageNo:   "F1785",
				From:       "TILBURY DOCK",
				To:         "SESOE",
			}},
		},
		{
			name:          "Forecast",
			vesselType:    "forecast",
			expectedCount: 1,
			expected: []models.Event{{
				VesselName: "ADELINE",
				Category:   "forecast",
				VoyageNo:   "A9999",
				From:       "NLVLI",
				To:         "FORDS JETTY",
			}},
		},
		{
			name:          "All",
			vesselType:    "all",
			expectedCount: 4,
			expected: []models.Event{{
				VesselName: "SILVER STURGEON", Category: "inport", VoyageNo: "S7670", Location: "WOODS QUAY"},
				{VesselName: "SAN NICOLAS MAERSK", Category: "arrivals", VoyageNo: "S7795", From: "MAPTM", To: "LONDON GATEWAY1"},
				{VesselName: "FRISIAN SPRING", Category: "departures", VoyageNo: "F1785", From: "TILBURY DOCK", To: "SESOE"},
				{VesselName: "ADELINE", Category: "forecast", VoyageNo: "A9999", From: "NLVLI", To: "FORDS JETTY"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vessels.ScrapeVessels(tc.vesselType)
			assert.NoError(t, err)
			assert.Len(t, result, tc.expectedCount)
			for i, expected := range tc.expected {
				actual := result[i]
				assert.Equal(t, expected.VesselName, actual.VesselName)
				assert.Equal(t, expected.Category, actual.Category)
				assert.Equal(t, expected.VoyageNo, actual.VoyageNo)
				assert.Equal(t, expected.From, actual.From)
				assert.Equal(t, expected.To, actual.To)
				assert.Equal(t, expected.Location, actual.Location)
			}
		})
	}
}

func TestScrapeVessels_ErrorCases(t *testing.T) {
	t.Run("MalformedJSON", func(t *testing.T) {
		// Create a server that returns malformed JSON.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{malformed json`))
		}))
		defer server.Close()

		originalURL := config.AppConfig.URLs.PortOfLondon
		config.AppConfig.URLs.PortOfLondon = server.URL
		defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

		_, err := vessels.ScrapeVessels("inport")
		assert.Error(t, err, "expected error for malformed JSON")
	})

	t.Run("MissingTimestamp", func(t *testing.T) {
		// Create a JSON response with missing timestamp for the inport vessel.
		response := `{
			"inport": [
				{
					"location_name": "WOODS QUAY",
					"vessel_name": "SILVER STURGEON",
					"visit": "S7670",
					"last_rep_dt": ""
				}
			],
			"arrivals": [],
			"departures": [],
			"forecast": []
		}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
		}))
		defer server.Close()

		originalURL := config.AppConfig.URLs.PortOfLondon
		config.AppConfig.URLs.PortOfLondon = server.URL
		defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

		vesselsResult, err := vessels.ScrapeVessels("inport")
		assert.NoError(t, err)
		// Ensure a vessel is returned.
		assert.Len(t, vesselsResult, 1)
		// Since missing timestamp triggers fallback using time.Now(), we only check that Timestamp is not zero.
		assert.False(t, vesselsResult[0].Timestamp.IsZero(), "expected fallback timestamp to be set")
	})
}
