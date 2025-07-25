package http

import (
	"os"
	"weather-api/internal/services/weather"
	"weather-api/pkg/observe"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

type routes struct {
	service *weather.WeatherService
	l       *observe.Logger
}

func NewRouter(
	app *fiber.App,
	weatherService *weather.WeatherService,
	l *observe.Logger,
) {
	r := &routes{
		service: weatherService,
		l:       l,
	}

	// Swagger documentation
	app.Get("/swagger/doc.json", func(c *fiber.Ctx) error {
		// Read the generated swagger.json file
		swaggerData, err := os.ReadFile("docs/swagger.json")
		if err != nil {
			return c.Status(fiber.ErrInternalServerError.Code).JSON(fiber.Map{"error": "Failed to read Swagger documentation"})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(swaggerData)
	})

	app.Get("/swagger/*", swagger.New(swagger.Config{
		URL:         "/swagger/doc.json",
		DeepLinking: true,
	}))

	// API routes
	app.Get("/weather", r.handleWeatherCall)
}
