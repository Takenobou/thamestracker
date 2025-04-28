package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/stretchr/testify/assert"
)

// --- Fake HTTP Client for vessels testing ---

type fakeHTTPClient struct {
	response *http.Response
	err      error
}

func (f *fakeHTTPClient) Get(url string) (*http.Response, error) {
	return f.response, f.err
}

// --- Fake in-memory cache implementation ---

type fakeCache struct {
	store map[string][]byte
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: make(map[string][]byte)}
}

func (f *fakeCache) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.store[key] = data
	return nil
}

func (f *fakeCache) Get(key string, dest interface{}) error {
	data, ok := f.store[key]
	if !ok {
		return errors.New("cache miss")
	}
	return json.Unmarshal(data, dest)
}

// --- Service Tests ---

// TestService_GetBridgeLifts_CacheMiss tests that on a cache miss, the service uses the bridge scraper.
func TestService_GetBridgeLifts_CacheMiss(t *testing.T) {
	// Prepare a fake HTML response for bridge lifts.
	sampleHTML := `
	<html>
		<body>
			<table>
				<tbody>
					<tr>
						<td>Sat</td>
						<td><time datetime="2025-04-05T00:00:00Z">05 Apr 2025</time></td>
						<td><time datetime="2025-04-05T17:45:00Z">17:45</time></td>
						<td>Paddle Steamer Dixie Queen</td>
						<td>Up river</td>
					</tr>
				</tbody>
			</table>
		</body>
	</html>
	`
	// Create a fake server that returns the sample HTML.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleHTML))
	}))
	defer server.Close()

	// Set the TowerBridge URL to our fake server's URL.
	originalURL := config.AppConfig.URLs.TowerBridge
	config.AppConfig.URLs.TowerBridge = server.URL
	defer func() { config.AppConfig.URLs.TowerBridge = originalURL }()

	// Use a fake cache that's initially empty.
	fc := newFakeCache()

	// For bridge scraping, Colly uses its own HTTP client so we can pass the default client.
	svc := service.NewService(httpclient.DefaultClient, fc)
	lifts, err := svc.GetBridgeLifts()
	assert.NoError(t, err)
	assert.Len(t, lifts, 1)
	assert.Equal(t, "2025-04-05", lifts[0].Date)
	assert.Equal(t, "17:45", lifts[0].Time)

	// Verify that the cache now contains the value.
	var cachedLifts []models.BridgeLift
	err = fc.Get("bridge_lifts", &cachedLifts)
	assert.NoError(t, err)
	assert.Len(t, cachedLifts, 1)
}

// TestService_GetVessels_CacheBehavior verifies that GetVessels fetches data on a cache miss
// and returns cached data on subsequent calls.
func TestService_GetVessels_CacheBehavior(t *testing.T) {
	// Prepare a fake JSON response for vessels.
	mockAPIResponse := `{
		"inport": [
			{
				"location_name": "WOODS QUAY",
				"vessel_name": "SILVER STURGEON",
				"visit": "S7670",
				"last_rep_dt": "2025-01-25 20:33:47.150"
			}
		],
		"arrivals": [],
		"departures": [],
		"forecast": []
	}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(mockAPIResponse))),
	}
	fakeClient := &fakeHTTPClient{response: resp, err: nil}
	originalClient := httpclient.DefaultClient
	httpclient.DefaultClient = fakeClient
	defer func() { httpclient.DefaultClient = originalClient }()

	// Override the PortOfLondon URL in config.
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = "http://fake-vessel-url"
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

	fc := newFakeCache()
	svc := service.NewService(fakeClient, fc)

	// First call: cache miss, data should be fetched and then cached.
	vessels1, err := svc.GetVessels("inport")
	assert.NoError(t, err)
	assert.Len(t, vessels1, 1)
	assert.Equal(t, "SILVER STURGEON", vessels1[0].Name)

	// Simulate a cache hit: clear fakeClient response to force error if called.
	fakeClient.response = nil
	vessels2, err := svc.GetVessels("inport")
	assert.NoError(t, err)
	assert.Len(t, vessels2, 1)
	assert.Equal(t, "SILVER STURGEON", vessels2[0].Name)
}

// TestService_GetVessels_ErrorHandling simulates a network error.
func TestService_GetVessels_ErrorHandling(t *testing.T) {
	// Create a fake HTTP client that returns an error.
	fakeClient := &fakeHTTPClient{response: nil, err: errors.New("network error")}
	originalClient := httpclient.DefaultClient
	httpclient.DefaultClient = fakeClient
	defer func() { httpclient.DefaultClient = originalClient }()

	// Override config.
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = "http://fake-vessel-url"
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }()

	fc := newFakeCache()
	svc := service.NewService(fakeClient, fc)
	_, err := svc.GetVessels("inport")
	assert.Error(t, err, "expected error when HTTP client fails")
}

// TestCircuitBreakerTrips verifies that after MaxFailures consecutive Get errors, the circuit breaker opens.
func TestCircuitBreakerTrips(t *testing.T) {
	// fake client that always fails
	callCount := 0
	fake := struct{ httpclient.Client }{}
	fake.Client = httpclient.ClientFunc(func(url string) (*http.Response, error) {
		callCount++
		return nil, fmt.Errorf("network fail %d", callCount)
	})
	// small breaker: trips after 2 failures with 1s cool-off
	breaker := httpclient.NewBreakerClient(fake, 2, 1)

	// first two attempts: underlying called and errors
	_, err := breaker.Get("test-url")
	assert.Error(t, err)
	_, err = breaker.Get("test-url")
	assert.Error(t, err)
	// third attempt: breaker is open, should not call underlying
	prevCalls := callCount
	_, err = breaker.Get("test-url")
	assert.Error(t, err)
	assert.True(t, callCount == prevCalls, "underlying client should not be called when breaker is open")
	assert.Contains(t, err.Error(), "circuit breaker is open")
}
