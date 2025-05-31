package vessels_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/stretchr/testify/assert"
)

func init() {
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

func TestScrapeVessels_AllTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockAPIResponse))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL

	types := []string{"inport", "arrivals", "departures", "forecast", "all"}
	for _, typ := range types {
		events, err := vessels.ScrapeVessels(typ)
		assert.NoError(t, err, typ)
		assert.NotEmpty(t, events, typ)
	}
}

func TestScrapeVessels_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	_, err := vessels.ScrapeVessels("inport")
	assert.Error(t, err)
}

func TestScrapeVessels_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	_, err := vessels.ScrapeVessels("inport")
	assert.Error(t, err)
}

func TestScrapeVessels_MissingVesselName(t *testing.T) {
	badJSON := `{"inport":[{"location_name":"WOODS QUAY","visit":"S7670","last_rep_dt":"2025-01-25 20:33:47.150"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(badJSON))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	events, err := vessels.ScrapeVessels("inport")
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestScrapeVessels_MissingVoyageNo(t *testing.T) {
	badJSON := `{"inport":[{"location_name":"WOODS QUAY","vessel_name":"SILVER STURGEON","last_rep_dt":"2025-01-25 20:33:47.150"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(badJSON))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	events, err := vessels.ScrapeVessels("inport")
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestScrapeVessels_InvalidType(t *testing.T) {
	config.AppConfig.URLs.PortOfLondon = "http://example.com"
	_, err := vessels.ScrapeVessels("notatype")
	assert.Error(t, err)
}

func TestScrapeVessels_EmptyLists(t *testing.T) {
	emptyJSON := `{"inport":[],"arrivals":[],"departures":[],"forecast":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(emptyJSON))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	events, err := vessels.ScrapeVessels("all")
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestScrapeVessels_TimestampFallback(t *testing.T) {
	badTimeJSON := `{"inport":[{"location_name":"WOODS QUAY","vessel_name":"SILVER STURGEON","visit":"S7670","last_rep_dt":"badtime"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(badTimeJSON))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL
	events, err := vessels.ScrapeVessels("inport")
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	// Should fallback to now, so timestamp is close to now
	delta := time.Since(events[0].Timestamp)
	assert.Less(t, delta.Seconds(), 5.0)
}

func BenchmarkScrapeVessels(b *testing.B) {
	// Generate a large mock response with 500 vessel entries
	var vesselsJSON = `{"inport": [`
	for i := 0; i < 500; i++ {
		vesselsJSON += `{"location_name":"WOODS QUAY","vessel_name":"VESSEL` + fmt.Sprint(i) + `","visit":"S` + fmt.Sprint(i) + `","last_rep_dt":"2025-01-25 20:33:47.150"}`
		if i < 499 {
			vesselsJSON += ","
		}
	}
	vesselsJSON += `],"arrivals":[],"departures":[],"forecast":[]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(vesselsJSON))
	}))
	defer server.Close()
	config.AppConfig.URLs.PortOfLondon = server.URL

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vessels.ScrapeVessels("inport")
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
