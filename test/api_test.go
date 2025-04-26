package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gofiber/fiber/v2"
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
func (f fakeService) HealthCheck() error { return nil }

// failingService implements ServiceInterface with a failing HealthCheck.
type failingService struct{}

func (f failingService) GetBridgeLifts() ([]models.BridgeLift, error)          { return nil, nil }
func (f failingService) GetVessels(vesselType string) ([]models.Vessel, error) { return nil, nil }
func (f failingService) HealthCheck() error                                    { return fmt.Errorf("unhealthy") }

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

func TestCalendarEndpoint(t *testing.T) {
	svc := fakeService{}
	handler := api.NewAPIHandler(svc)
	app := fiber.New()
	api.SetupRoutes(app, handler)

	req := httptest.NewRequest(http.MethodGet, "/calendar.ics", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	assert.Contains(t, body, "BEGIN:VCALENDAR")
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
