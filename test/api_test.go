package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

// fakeService implements the ServiceInterface for testing.
type fakeService struct{}

func (f fakeService) GetBridgeLifts() ([]models.BridgeLift, error) {
	return []models.BridgeLift{
		{Date: "2025-04-05", Time: "17:45", Vessel: "Fake Bridge Lift", Direction: "Up river"},
	}, nil
}

func (f fakeService) GetVessels(vesselType string) ([]models.Vessel, error) {
	return []models.Vessel{
		{Time: "20:33", Date: "25/01/2025", LocationName: "Fake Port", Name: "Fake Vessel", VoyageNo: "F123", Type: vesselType},
	}, nil
}

// Add HealthCheck to fakeService for healthz endpoint testing
func (f fakeService) HealthCheck(ctx context.Context) error { return nil }

// Add ListLocations to fakeService
func (f fakeService) ListLocations() ([]service.LocationStats, error) {
	return []service.LocationStats{
		{Name: "PortA", Code: "", Inport: 1, Arrivals: 2, Departures: 3, Total: 6},
		{Name: "PortB", Code: "", Inport: 0, Arrivals: 1, Departures: 0, Total: 1},
	}, nil
}

// Add GetFilteredVessels to fakeService
func (f fakeService) GetFilteredVessels(vesselType, location string) ([]models.Vessel, error) {
	return f.GetVessels(vesselType)
}

// failingService implements ServiceInterface with a failing HealthCheck.
type failingService struct{}

func (f failingService) GetBridgeLifts() ([]models.BridgeLift, error)          { return nil, nil }
func (f failingService) GetVessels(vesselType string) ([]models.Vessel, error) { return nil, nil }
func (f failingService) HealthCheck(ctx context.Context) error                 { return fmt.Errorf("unhealthy") }
func (f failingService) ListLocations() ([]service.LocationStats, error) {
	return nil, fmt.Errorf("fail")
}

// Add GetFilteredVessels to failingService
func (f failingService) GetFilteredVessels(vesselType, location string) ([]models.Vessel, error) {
	return f.GetVessels(vesselType)
}

// fake service that simulates circuit breaker open
type openBreakerService struct{}

func (s openBreakerService) GetBridgeLifts() ([]models.BridgeLift, error) {
	return nil, gobreaker.ErrOpenState
}
func (s openBreakerService) GetVessels(vesselType string) ([]models.Vessel, error) {
	return nil, gobreaker.ErrOpenState
}
func (s openBreakerService) GetFilteredVessels(vesselType, location string) ([]models.Vessel, error) {
	return nil, gobreaker.ErrOpenState
}
func (s openBreakerService) HealthCheck(ctx context.Context) error {
	return gobreaker.ErrOpenState
}
func (s openBreakerService) ListLocations() ([]service.LocationStats, error) {
	return nil, gobreaker.ErrOpenState
}

func TestGetBridgeLiftsEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/bridge-lifts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var lifts []models.BridgeLift
	err = json.NewDecoder(resp.Body).Decode(&lifts)
	assert.NoError(t, err)
	assert.Len(t, lifts, 1)
	assert.Equal(t, "Fake Bridge Lift", lifts[0].Vessel)
}

func TestGetBridgeLifts_CircuitBreakerOpen(t *testing.T) {
	handler := api.NewAPIHandler(openBreakerService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/bridge-lifts", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	retry := resp.Header.Get("Retry-After")
	assert.NotEmpty(t, retry)
}

func TestGetVesselsEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/vessels?type=inport", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var vessels []models.Vessel
	err = json.NewDecoder(resp.Body).Decode(&vessels)
	assert.NoError(t, err)
	assert.Len(t, vessels, 1)
	assert.Equal(t, "Fake Vessel", vessels[0].Name)
}

func TestGetVessels_CircuitBreakerOpen(t *testing.T) {
	handler := api.NewAPIHandler(openBreakerService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/vessels?type=inport", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	retry := resp.Header.Get("Retry-After")
	assert.NotEmpty(t, retry)
}

func TestGetVessels_InvalidType(t *testing.T) {
	handler := api.NewAPIHandler(fakeService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/vessels?type=invalid", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"], "invalid type")
}

func TestGetVessels_InvalidAfterBefore(t *testing.T) {
	handler := api.NewAPIHandler(fakeService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	// invalid after
	req1 := httptest.NewRequest(http.MethodGet, "/vessels?type=inport&after=bad", nil)
	resp1, _ := app.Test(req1)
	assert.Equal(t, http.StatusBadRequest, resp1.StatusCode)

	// invalid before
	req2 := httptest.NewRequest(http.MethodGet, "/vessels?type=inport&before=bad", nil)
	resp2, _ := app.Test(req2)
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
}

func TestBridgeCalendarEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/bridge-lifts/calendar.ics", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	body := buf.String()
	assert.Contains(t, body, "BEGIN:VCALENDAR")
}

func TestBridgeCalendar_CircuitBreakerOpen(t *testing.T) {
	handler := api.NewAPIHandler(openBreakerService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/bridge-lifts/calendar.ics", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	retry := resp.Header.Get("Retry-After")
	assert.NotEmpty(t, retry)
}

func TestVesselsCalendarEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/vessels/calendar.ics?type=inport", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	body := buf.String()
	assert.Contains(t, body, "BEGIN:VCALENDAR")
}

func TestVesselsCalendar_CircuitBreakerOpen(t *testing.T) {
	handler := api.NewAPIHandler(openBreakerService{})
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/vessels/calendar.ics?type=inport", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	retry := resp.Header.Get("Retry-After")
	assert.NotEmpty(t, retry)
}

func TestHealthzEndpoint_Success(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestHealthzEndpoint_Failure(t *testing.T) {
	svc := failingService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "fail", result["status"])
	assert.Contains(t, result["error"], "unhealthy")
}

func TestVesselsJSONAndICACountParity(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	// Define query parameters to test
	// test various vessel types and unique filter
	queries := []string{
		"?type=inport",
		"?type=arrivals",
		"?type=departures",
		"?type=forecast",
		"?unique=true",
	}
	for _, q := range queries {
		// JSON endpoint
		reqJSON := httptest.NewRequest(http.MethodGet, "/vessels"+q, nil)
		respJSON, err := app.Test(reqJSON)
		assert.NoError(t, err)
		var vessels []models.Vessel
		err = json.NewDecoder(respJSON.Body).Decode(&vessels)
		assert.NoError(t, err)
		countJSON := len(vessels)

		// ICS endpoint: ensure type param present (default to all)
		icsQuery := q
		if !strings.Contains(q, "type=") {
			// no type in query, default to all
			icsQuery = "?type=all&" + strings.TrimPrefix(q, "?")
		}
		reqICS := httptest.NewRequest(http.MethodGet, "/vessels/calendar.ics"+icsQuery, nil)
		respICS, err := app.Test(reqICS)
		assert.NoError(t, err)
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(respICS.Body)
		icsBody := buf.String()
		countICS := strings.Count(icsBody, "BEGIN:VEVENT")

		assert.Equal(t, countJSON, countICS, fmt.Sprintf("mismatch for query '%s'", q))
	}
}

// TestGetLocationsEndpoint tests the /locations endpoint with no filters.
func TestGetLocationsEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var locs []service.LocationStats
	err = json.NewDecoder(resp.Body).Decode(&locs)
	assert.NoError(t, err)
	assert.Len(t, locs, 2)
	assert.Equal(t, "PortA", locs[0].Name)
	assert.Equal(t, 6, locs[0].Total)
}

// TestGetLocationsFilters tests filtering by minTotal and q.
func TestGetLocationsFilters(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	// minTotal filter
	req1 := httptest.NewRequest(http.MethodGet, "/locations?minTotal=5", nil)
	resp1, _ := app.Test(req1)
	var locs1 []service.LocationStats
	_ = json.NewDecoder(resp1.Body).Decode(&locs1)
	assert.Len(t, locs1, 1)
	assert.Equal(t, "PortA", locs1[0].Name)

	// q filter
	req2 := httptest.NewRequest(http.MethodGet, "/locations?q=portb", nil)
	resp2, _ := app.Test(req2)
	var locs2 []service.LocationStats
	_ = json.NewDecoder(resp2.Body).Decode(&locs2)
	assert.Len(t, locs2, 1)
	assert.Equal(t, "PortB", locs2[0].Name)
}
