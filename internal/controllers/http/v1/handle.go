package http

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultForecastWindow = 5
	maxForecastWindow     = 5
	maxLatitude           = 90
	maxLongitude          = 180
	minLatitude           = -90
	minLongitude          = -180
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Missing required parameter: lat"`
}

// GetWeatherForecast godoc
// @Summary Get weather forecast
// @Description Retrieves weather forecast data for a specific location from multiple providers
// @Tags Weather
// @Accept json
// @Produce json
// @Param lat query number true "Lat coordinate (-90 to 90)" minimum(-90) maximum(90) example(40.7128)
// @Param lon query number true "Lon coordinate (-180 to 180)" minimum(-180) maximum(180) example(-74.006)
// @Param days query integer false "Number of forecast days (1-14, default: 5)" minimum(1) maximum(14) example(3)
// @Success 200 {object} WeatherResponse "Successful response"
// @Failure 400 {object} ErrorResponse "Bad request - invalid parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /weather [get]
// @Example {curl} Example usage:
//
//	curl -X GET "http://localhost:8080/weather?lat=40.7128&lon=-74.006&days=3"
func (r *routes) handleWeatherCall(c *fiber.Ctx) error {
	lat, lon, forecastWindow, err := validateParameters(c)
	if err != nil {
		r.l.Error(err, map[string]any{
			"lat":            c.Query("lat"),
			"lon":            c.Query("lon"),
			"forecastWindow": c.Query("days"),
		})

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: err.Error(),
		})
	}

	forecasts, err := r.service.FetchForecasts(c.Context(), lat, lon, forecastWindow)
	if err != nil {
		r.l.Error(err, map[string]any{
			"lat":            lat,
			"lon":            lon,
			"forecastWindow": forecastWindow,
		})

		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to fetch weather data",
		})
	}

	return c.JSON(forecasts)
}

func validateParameters(c *fiber.Ctx) (float64, float64, int, error) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	if latStr == "" {
		return 0, 0, 0, fmt.Errorf("missing required parameter: lat")
	}

	if lonStr == "" {
		return 0, 0, 0, fmt.Errorf("missing required parameter: lon")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid latitude format: %s", latStr)
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid longitude format: %s", lonStr)
	}

	// Validate latitude and longitude ranges
	if lat < minLatitude || lat > maxLatitude {
		return 0, 0, 0, fmt.Errorf("latitude must be between %d and %d, got: %f", minLatitude, maxLatitude, lat)
	}
	if lon < minLongitude || lon > maxLongitude {
		return 0, 0, 0, fmt.Errorf("longitude must be between %d and %d, got: %f", minLongitude, maxLongitude, lon)
	}

	// Optional: Validate forecast window if provided
	daysStr := c.Query("days")
	days := defaultForecastWindow
	if daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid days parameter: %s", daysStr)
		}
		if days < 1 || days > maxForecastWindow {
			return 0, 0, 0, fmt.Errorf("days must be between 1 and %d", maxForecastWindow)
		}
	}

	return lat, lon, days, nil
}
