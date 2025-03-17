package api

import "github.com/gofiber/fiber/v2"

// SetupRoutes initialises routes using the provided API handler.
func SetupRoutes(app *fiber.App, handler *APIHandler) {
	app.Get("/bridge-lifts", handler.GetBridgeLifts)
	app.Get("/ships", handler.GetShips)
	app.Get("/calendar.ics", handler.CalendarHandler)
}
