package api

import "github.com/gofiber/fiber/v2"

func SetupRoutes(app *fiber.App) {
	app.Get("/bridge-lifts", GetBridgeLifts)
	app.Get("/ships", GetShips)
	app.Get("/arrivals", GetArrivals)
	app.Get("/departures", GetDepartures)
	app.Get("/forecast", GetForecast)
}
