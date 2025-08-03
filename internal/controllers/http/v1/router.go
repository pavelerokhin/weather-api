package http

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"weather-api/internal/services/weather"
	"weather-api/pkg/logger"
)

type routes struct {
	service *weather.WeatherService
	l       *logger.Logger
}

func NewRouter(
	app *fiber.App,
	weatherService *weather.WeatherService,
	l *logger.Logger,
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
