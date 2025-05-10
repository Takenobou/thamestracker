package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	importedLogger "github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

// fakeService implements the ServiceInterface for testing.
type fakeService struct{}

func (f fakeService) GetBridgeLifts() ([]models.Event, error) {
	return []models.Event{
		{
			Timestamp:  time.Date(2025, 4, 5, 17, 45, 0, 0, time.UTC),
			VesselName: "Fake Bridge Lift",
			Category:   "bridge",
			Direction:  "Up river",
			Location:   "Tower Bridge Road, London",
		},
	}, nil
}

func (f fakeService) GetVessels(vesselType string) ([]models.Event, error) {
	return []models.Event{
		{
			Timestamp:  time.Date(2025, 1, 25, 20, 33, 0, 0, time.UTC),
			VesselName: "Fake Vessel",
			Category:   vesselType,
			VoyageNo:   "F123",
			Location:   "Fake Port",
		},
	}, nil
}

func (f fakeService) HealthCheck(ctx context.Context) error { return nil }

func (f fakeService) ListLocations() ([]service.LocationStats, error) {
	return []service.LocationStats{
		{Name: "PortA", Code: "", Inport: 1, Arrivals: 2, Departures: 3, Total: 6},
		{Name: "PortB", Code: "", Inport: 0, Arrivals: 1, Departures: 0, Total: 1},
	}, nil
}

func (f fakeService) GetFilteredVessels(vesselType, location string) ([]models.Event, error) {
	return f.GetVessels(vesselType)
}

// Error fakes for testing

type errorService struct {
	bridgeErr error
	vesselErr error
}

func (e errorService) GetBridgeLifts() ([]models.Event, error)   { return nil, e.bridgeErr }
func (e errorService) GetVessels(string) ([]models.Event, error) { return nil, e.vesselErr }
func (e errorService) GetFilteredVessels(string, string) ([]models.Event, error) {
	return nil, e.vesselErr
}
func (e errorService) HealthCheck(ctx context.Context) error           { return nil }
func (e errorService) ListLocations() ([]service.LocationStats, error) { return nil, nil }

func setupTestApp(svc ServiceInterface) *fiber.App {
	h := NewAPIHandler(svc)
	app := fiber.New()
	app.Get("/bridge-lifts", h.GetBridgeLifts)
	app.Get("/vessels", h.GetVessels)
	app.Get("/bridge-lifts/calendar.ics", h.BridgeCalendarHandler)
	app.Get("/vessels/calendar.ics", h.VesselsCalendarHandler)
	return app
}

func TestBridgeLifts_ErrorPropagation(t *testing.T) {
	app := setupTestApp(errorService{bridgeErr: errors.New("fail")})
	r := httptest.NewRequest(http.MethodGet, "/bridge-lifts", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestBridgeLifts_CircuitBreaker503(t *testing.T) {
	app := setupTestApp(errorService{bridgeErr: gobreaker.ErrOpenState})
	r := httptest.NewRequest(http.MethodGet, "/bridge-lifts", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 503, resp.StatusCode)
}

func TestBridgeCalendar_ErrorPropagation(t *testing.T) {
	app := setupTestApp(errorService{bridgeErr: errors.New("fail")})
	r := httptest.NewRequest(http.MethodGet, "/bridge-lifts/calendar.ics", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestBridgeCalendar_CircuitBreaker503(t *testing.T) {
	app := setupTestApp(errorService{bridgeErr: gobreaker.ErrOpenState})
	r := httptest.NewRequest(http.MethodGet, "/bridge-lifts/calendar.ics", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 503, resp.StatusCode)
}

func TestVessels_ErrorPropagation(t *testing.T) {
	app := setupTestApp(errorService{vesselErr: errors.New("fail")})
	r := httptest.NewRequest(http.MethodGet, "/vessels?type=all", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestVessels_CircuitBreaker503(t *testing.T) {
	app := setupTestApp(errorService{vesselErr: gobreaker.ErrOpenState})
	r := httptest.NewRequest(http.MethodGet, "/vessels?type=all", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 503, resp.StatusCode)
}

func TestVesselsCalendar_ErrorPropagation(t *testing.T) {
	app := setupTestApp(errorService{vesselErr: errors.New("fail")})
	r := httptest.NewRequest(http.MethodGet, "/vessels/calendar.ics?type=all", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestVesselsCalendar_CircuitBreaker503(t *testing.T) {
	app := setupTestApp(errorService{vesselErr: gobreaker.ErrOpenState})
	r := httptest.NewRequest(http.MethodGet, "/vessels/calendar.ics?type=all", nil)
	resp, _ := app.Test(r)
	assert.Equal(t, 503, resp.StatusCode)
}

func TestAPI_PackageLoads(t *testing.T) {
	assert.True(t, true)
}

func TestMain(m *testing.M) {
	// Ensure logger is initialized for all tests
	importedLogger.InitLogger()
	os.Exit(m.Run())
}
