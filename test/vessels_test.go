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
	// Create a mock HTTP server for valid JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockAPIResponse))
	}))
	defer server.Close()

	// Override API URL.
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = server.URL
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

	// Table-driven tests for valid responses.
	tests := []struct {
		name            string
		vesselType      string
		expectedCount   int
		expectedDetails []models.Vessel
	}{
		{
			name:          "Inport",
			vesselType:    "inport",
			expectedCount: 1,
			expectedDetails: []models.Vessel{
				{
					Time:         "20:33",
					Date:         "25/01/2025",
					LocationName: "WOODS QUAY",
					Name:         "SILVER STURGEON",
					VoyageNo:     "S7670",
					Type:         "inport",
				},
			},
		},
		{
			name:          "Arrivals",
			vesselType:    "arrivals",
			expectedCount: 1,
			expectedDetails: []models.Vessel{
				{
					Time:         "14:22",
					Date:         "13/03/2025",
					LocationFrom: "MAPTM",
					LocationTo:   "LONDON GATEWAY1",
					Name:         "SAN NICOLAS MAERSK",
					VoyageNo:     "S7795",
					Type:         "arrivals",
				},
			},
		},
		{
			name:          "Departures",
			vesselType:    "departures",
			expectedCount: 1,
			expectedDetails: []models.Vessel{
				{
					Time:         "15:39",
					Date:         "13/03/2025",
					LocationFrom: "TILBURY DOCK",
					LocationTo:   "SESOE",
					Name:         "FRISIAN SPRING",
					VoyageNo:     "F1785",
					Type:         "departures",
				},
			},
		},
		{
			name:          "Forecast",
			vesselType:    "forecast",
			expectedCount: 1,
			expectedDetails: []models.Vessel{
				{
					Time:         "14:15",
					Date:         "14/03/2025",
					LocationFrom: "NLVLI",
					LocationTo:   "FORDS JETTY",
					Name:         "ADELINE",
					VoyageNo:     "A9999",
					Type:         "forecast",
				},
			},
		},
		{
			name:          "All",
			vesselType:    "all",
			expectedCount: 4,
			expectedDetails: []models.Vessel{
				{
					Time:         "20:33",
					Date:         "25/01/2025",
					LocationName: "WOODS QUAY",
					Name:         "SILVER STURGEON",
					VoyageNo:     "S7670",
					Type:         "inport",
				},
				{
					Time:         "14:22",
					Date:         "13/03/2025",
					LocationFrom: "MAPTM",
					LocationTo:   "LONDON GATEWAY1",
					Name:         "SAN NICOLAS MAERSK",
					VoyageNo:     "S7795",
					Type:         "arrivals",
				},
				{
					Time:         "15:39",
					Date:         "13/03/2025",
					LocationFrom: "TILBURY DOCK",
					LocationTo:   "SESOE",
					Name:         "FRISIAN SPRING",
					VoyageNo:     "F1785",
					Type:         "departures",
				},
				{
					Time:         "14:15",
					Date:         "14/03/2025",
					LocationFrom: "NLVLI",
					LocationTo:   "FORDS JETTY",
					Name:         "ADELINE",
					VoyageNo:     "A9999",
					Type:         "forecast",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vesselsResult, err := vessels.ScrapeVessels(tc.vesselType)
			assert.NoError(t, err)
			assert.Len(t, vesselsResult, tc.expectedCount, "unexpected number of vessels for type %s", tc.vesselType)

			// Validate each expected vessel's key fields.
			for i, expected := range tc.expectedDetails {
				actual := vesselsResult[i]
				assert.Equal(t, expected.Time, actual.Time, "Time mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.Date, actual.Date, "Date mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.LocationFrom, actual.LocationFrom, "LocationFrom mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.LocationTo, actual.LocationTo, "LocationTo mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.LocationName, actual.LocationName, "LocationName mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.Name, actual.Name, "Name mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.VoyageNo, actual.VoyageNo, "VoyageNo mismatch for vessel %s", expected.Name)
				assert.Equal(t, expected.Type, actual.Type, "Type mismatch for vessel %s", expected.Name)
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
		// Since missing timestamp triggers fallback using time.Now(), we only check that Time is not empty.
		assert.NotEmpty(t, vesselsResult[0].Time, "expected fallback time to be set")
	})
}
