package api

import "github.com/gofiber/fiber/v2"

func SetupRoutes(app *fiber.App) {
	app.Get("/bridge-lifts", GetBridgeLifts)
	app.Get("/rare-lifts", GetRareLifts)
	app.Get("/ships", GetShips)
}
