package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"weather-api/internal/models"
	"weather-api/pkg/logger"
)

const (
	OpenMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"
)

type OpenMeteoRepository struct {
	httpClient HTTPClient
	l          *logger.Logger
}

func NewOpenMeteoRepository(l *logger.Logger, httpClient HTTPClient) *OpenMeteoRepository {
	return &OpenMeteoRepository{
		httpClient: httpClient,
		l:          l,
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

func (o *OpenMeteoRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) (models.Forecast, error) {
	forecast := models.Forecast{
		RepositoryName: o.Name(),
		Lat:            lat,
		Lon:            lon,
		ForecastWindow: forecastWindow,
	}

	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min&forecast_days=%d&timezone=auto", OpenMeteoBaseURL, lat, lon, forecastWindow)

	o.l.Info("making openmeteo API request", map[string]any{
		"params": forecast.RequestParams(),
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return forecast, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return forecast, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	o.l.Info("received openmeteo API response", map[string]any{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return forecast, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP error status codes
	if resp.StatusCode != http.StatusOK {
		return forecast, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		Daily OpenMeteoResponse `json:"daily"`
	}

	if err = json.Unmarshal(body, &response); err != nil {
		return forecast, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	o.l.Info("parsed API response", map[string]any{
		"days": len(response.Daily.Time),
	})

	// Validate that we have forecast data
	if len(response.Daily.Time) == 0 {
		return forecast, fmt.Errorf("no forecast data available")
	}

	// Convert API response to weather forecast models
	forecastData, err := dailyTemperaturesOpenMeteo(response.Daily)
	if err != nil {
		return forecast, fmt.Errorf("failed to build forecast: %w", err)
	}

	forecast.ForecastData = forecastData

	return forecast, nil
}

// buildForecastFromResponse converts the API response to weather forecast models
func dailyTemperaturesOpenMeteo(daily OpenMeteoResponse) ([]models.WeatherData, error) {
	var forecastDays []models.WeatherData

	// Find the minimum length to avoid index out of bounds
	minLength := min(len(daily.Time), len(daily.Temperature2mMax), len(daily.Temperature2mMin))

	// Build forecast for each day
	for i := 0; i < minLength; i++ {
		dayForecast, err := createDayForecast(daily, i)
		if err != nil {
			return nil, err
		}

		forecastDays = append(forecastDays, *dayForecast)
	}

	return forecastDays, nil
}

// createDayForecast creates a single day forecast, validating temperature data
func createDayForecast(daily OpenMeteoResponse, index int) (*models.WeatherData, error) {
	maxTemp := daily.Temperature2mMax[index]
	minTemp := daily.Temperature2mMin[index]

	// Parse the date string
	date, err := time.Parse("2006-01-02", daily.Time[index])
	if err != nil {
		return nil, fmt.Errorf("failed to parse date %s: %w", daily.Time[index], err)
	}

	return &models.WeatherData{
		Date:    &date,
		TempMax: maxTemp,
		TempMin: minTemp,
	}, nil
}
