package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"weather-api/internal/models"
	"weather-api/pkg/observe"
)

const (
	OpenMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"
)

type OpenMeteoRepository struct {
	l *observe.Logger
}

func NewOpenMeteoRepository(l *observe.Logger) *OpenMeteoRepository {
	return &OpenMeteoRepository{
		l: l,
	}
}

func (o *OpenMeteoRepository) Name() string {
	return "open-meteo"
}

type OpenMeteoResponse struct {
	Time             []string  `json:"time"`
	Temperature2mMax []float64 `json:"temperature_2m_max"`
	Temperature2mMin []float64 `json:"temperature_2m_min"`
}

type OpenMeteoErrorResponse struct {
	Error   bool   `json:"error"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type DailyWeatherData struct {
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
}

func (o *OpenMeteoRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error) {
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min&forecast_days=%d&timezone=auto", OpenMeteoBaseURL, lat, lon, forecastWindow)

	o.l.Info("making API request", map[string]any{
		"repository": o.Name(),
		"url":        url,
		"lat":        lat,
		"lon":        lon,
		"window":     forecastWindow,
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		o.l.Error(err, map[string]any{
			"repository": o.Name(),
		})
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	o.l.Info("received API response", map[string]any{
		"repository": o.Name(),
		"status":     resp.StatusCode,
		"statusText": resp.Status,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.l.Error(err, map[string]any{
			"repository": o.Name(),
		})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP error status codes
	if resp.StatusCode != http.StatusOK {
		var errorResp OpenMeteoErrorResponse
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil && errorResp.Error {
			o.l.Error(fmt.Errorf("API error: %s", errorResp.Reason), map[string]any{
				"repository": o.Name(),
				"statusCode": resp.StatusCode,
				"reason":     errorResp.Reason,
				"message":    errorResp.Message,
			})
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorResp.Reason)
		}

		o.l.Error(fmt.Errorf("HTTP error: %s", resp.Status), map[string]any{
			"repository": o.Name(),
			"statusCode": resp.StatusCode,
			"body":       string(body),
		})
		return nil, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		Daily OpenMeteoResponse `json:"daily"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		o.l.Error(err, map[string]any{
			"repository": o.Name(),
			"body":       string(body),
		})
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	o.l.Info("parsed API response", map[string]any{
		"repository": o.Name(),
		"days":       len(response.Daily.Time),
	})

	// Validate that we have forecast data
	if !o.hasForecastData(response.Daily) {
		return nil, fmt.Errorf("no forecast data available")
	}

	// Convert API response to weather forecast models
	forecastDays := o.buildForecastFromResponse(response.Daily)

	o.l.Info("final forecast result", map[string]any{
		"repository": o.Name(),
		"days":       len(forecastDays),
		"forecast":   forecastDays,
	})

	return forecastDays, nil
}

// hasForecastData checks if the daily forecast data contains valid time entries
func (o *OpenMeteoRepository) hasForecastData(daily OpenMeteoResponse) bool {
	return len(daily.Time) > 0
}

// buildForecastFromResponse converts the API response to weather forecast models
func (o *OpenMeteoRepository) buildForecastFromResponse(daily OpenMeteoResponse) []models.Response {
	var forecastDays []models.Response

	// Find the minimum length to avoid index out of bounds
	minLength := min(len(daily.Time), len(daily.Temperature2mMax), len(daily.Temperature2mMin))

	// Check if we have enough data
	if minLength == 0 {
		o.l.Warning("insufficient temperature data", map[string]any{
			"repository": o.Name(),
			"timeLength": len(daily.Time),
			"maxLength":  len(daily.Temperature2mMax),
			"minLength":  len(daily.Temperature2mMin),
		})
		return nil
	}

	// Build forecast for each day
	for i := 0; i < minLength; i++ {
		dayForecast := o.createDayForecast(daily, i)
		if dayForecast != nil {
			forecastDays = append(forecastDays, *dayForecast)
		}
	}

	return forecastDays
}

// createDayForecast creates a single day forecast, validating temperature data
func (o *OpenMeteoRepository) createDayForecast(daily OpenMeteoResponse, index int) *models.Response {
	maxTemp := daily.Temperature2mMax[index]
	minTemp := daily.Temperature2mMin[index]

	// Validate temperature data (max should be >= min)
	if maxTemp < minTemp {
		o.l.Warning("invalid temperature data (max < min)", map[string]any{
			"repository": o.Name(),
			"date":       daily.Time[index],
			"max":        maxTemp,
			"min":        minTemp,
		})
		return nil
	}

	return &models.Response{
		Date:    daily.Time[index],
		TempMax: maxTemp,
		TempMin: minTemp,
	}
}
