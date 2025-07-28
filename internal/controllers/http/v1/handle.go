package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// WeatherResponse represents the weather forecast response
type WeatherResponse struct {
	Latitude       float64                  `json:"latitude" example:"40.7128"`
	Longitude      float64                  `json:"longitude" example:"-74.006"`
	ForecastWindow int                      `json:"forecast_window" example:"5"`
	Forecasts      map[string][]WeatherData `json:"forecasts"`
}

// WeatherData represents a single day's weather data
type WeatherData struct {
	Date    string  `json:"date" example:"2025-07-25"`
	TempMax float64 `json:"temp_max" example:"38.0"`
	TempMin float64 `json:"temp_min" example:"24.3"`
}

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
// @Param lat query number true "Latitude coordinate (-90 to 90)" minimum(-90) maximum(90) example(40.7128)
// @Param lon query number true "Longitude coordinate (-180 to 180)" minimum(-180) maximum(180) example(-74.006)
// @Param days query integer false "Number of forecast days (1-14, default: 5)" minimum(1) maximum(14) example(3)
// @Success 200 {object} WeatherResponse "Successful response"
// @Failure 400 {object} ErrorResponse "Bad request - invalid parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /weather [get]
// @Example {curl} Example usage:
//
//	curl -X GET "http://localhost:8080/weather?lat=40.7128&lon=-74.006&days=3"
func (r *routes) handleWeatherCall(c *fiber.Ctx) error {
	const defaultForecastWindow = 5
	const maxForecastWindow = 14

	lat := c.Query("lat")
	lon := c.Query("lon")

	// Check for required parameters
	if lat == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Missing required parameter: lat",
		})
	}

	if lon == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Missing required parameter: lon",
		})
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid latitude format",
		})
	}

	if latFloat < -90 || latFloat > 90 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Latitude must be between -90 and 90",
		})
	}

	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid longitude format",
		})
	}

	if lonFloat < -180 || lonFloat > 180 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Longitude must be between -180 and 180",
		})
	}

	// Get forecast window from query parameter (default to 5 days)
	forecastWindow := defaultForecastWindow
	if window := c.Query("days"); window != "" {
		if days, err := strconv.Atoi(window); err == nil && days > 0 && days <= maxForecastWindow {
			forecastWindow = days
		} else {
			// Log warning but continue with default value
			r.l.Warning("invalid days parameter, using default", map[string]any{
				"provided": window,
				"default":  forecastWindow,
			})
		}
	}

	forecasts, err := r.service.FetchForecasts(c.Context(), latFloat, lonFloat, forecastWindow)
	if err != nil {
		r.l.Error(err, map[string]any{
			"lat":            latFloat,
			"lon":            lonFloat,
			"forecastWindow": forecastWindow,
		})

		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to fetch weather data",
		})
	}

	// Convert forecasts to the documented response format
	weatherForecasts := make(map[string][]WeatherData)
	for provider, data := range forecasts {
		if data != nil {
			weatherForecasts[provider] = make([]WeatherData, len(data))
			for i, item := range data {
				weatherForecasts[provider][i] = WeatherData{
					Date:    item.Date,
					TempMax: item.TempMax,
					TempMin: item.TempMin,
				}
			}
		}
	}

	response := WeatherResponse{
		Latitude:       latFloat,
		Longitude:      lonFloat,
		ForecastWindow: forecastWindow,
		Forecasts:      weatherForecasts,
	}

	return c.JSON(response)
}
