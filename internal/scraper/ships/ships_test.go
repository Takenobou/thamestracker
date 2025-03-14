package ships

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/stretchr/testify/assert"
)

// Mock API response covering all ship categories
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

func TestScrapeShips(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockAPIResponse))
	}))
	defer server.Close()

	// Backup and override API URL
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = server.URL
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }() // Restore after test

	// Define test cases for different ship types
	tests := []struct {
		name     string
		shipType string
		expected models.Ship
	}{
		{
			name:     "Inport",
			shipType: "inport",
			expected: models.Ship{
				Time:         "20:33",
				Date:         "25/01/2025",
				LocationName: "WOODS QUAY",
				Name:         "SILVER STURGEON",
				VoyageNo:     "S7670",
				Type:         "inport",
			},
		},
		{
			name:     "Arrivals",
			shipType: "arrivals",
			expected: models.Ship{
				Time:         "14:22",
				Date:         "13/03/2025",
				LocationFrom: "MAPTM",
				LocationTo:   "LONDON GATEWAY1",
				Name:         "SAN NICOLAS MAERSK",
				VoyageNo:     "S7795",
				Type:         "arrivals",
			},
		},
		{
			name:     "Departures",
			shipType: "departures",
			expected: models.Ship{
				Time:         "15:39",
				Date:         "13/03/2025",
				LocationFrom: "TILBURY DOCK",
				LocationTo:   "SESOE",
				Name:         "FRISIAN SPRING",
				VoyageNo:     "F1785",
				Type:         "departures",
			},
		},
		{
			name:     "Forecast",
			shipType: "forecast",
			expected: models.Ship{
				Time:         "14:15",
				Date:         "14/03/2025",
				LocationFrom: "NLVLI",
				LocationTo:   "FORDS JETTY",
				Name:         "ADELINE",
				VoyageNo:     "A9999",
				Type:         "forecast",
			},
		},
	}

	// Run tests for each ship type
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ships, err := ScrapeShips(tc.shipType)
			assert.NoError(t, err)
			assert.Len(t, ships, 1)

			// Validate each field
			assert.Equal(t, tc.expected.Time, ships[0].Time, "Time mismatch")
			assert.Equal(t, tc.expected.Date, ships[0].Date, "Date mismatch")
			assert.Equal(t, tc.expected.LocationFrom, ships[0].LocationFrom, "LocationFrom mismatch")
			assert.Equal(t, tc.expected.LocationTo, ships[0].LocationTo, "LocationTo mismatch")
			assert.Equal(t, tc.expected.LocationName, ships[0].LocationName, "LocationName mismatch")
			assert.Equal(t, tc.expected.Name, ships[0].Name, "Name mismatch")
			assert.Equal(t, tc.expected.VoyageNo, ships[0].VoyageNo, "VoyageNo mismatch")
			assert.Equal(t, tc.expected.Type, ships[0].Type, "Type mismatch")
		})
	}
}
