package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"weather-api/internal/models"
	"weather-api/pkg/logger"
)

const (
	WeatherAPIBaseURL = "https://api.openweathermap.org/data/2.5/forecast"
)

type WeatherAPIRepository struct {
	APIKey     string
	httpClient HTTPClient
	l          *logger.Logger
}

func NewWeatherAPIRepository(apiKey string, l *logger.Logger, httpClient HTTPClient) (*WeatherAPIRepository, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("API key cannot be empty")
	}

	return &WeatherAPIRepository{
		APIKey:     apiKey,
		httpClient: httpClient,
		l:          l,
	}, nil
}

func (w *WeatherAPIRepository) Name() string {
	return "weatherapi"
}

type WeatherAPIResponse struct {
	List []struct {
		Dt    int64  `json:"dt"`
		DtTxt string `json:"dt_txt"`
		Main  struct {
			TempMin float64 `json:"temp_min"`
			TempMax float64 `json:"temp_max"`
		} `json:"main"`
	} `json:"list"`
}

func (w *WeatherAPIRepository) FetchForecast(
	ctx context.Context,
	lat float64,
	lon float64,
	forecastWindow int,
) (models.Forecast, error) {
	forecast := models.Forecast{
		RepositoryName: w.Name(),
		Lat:            lat,
		Lon:            lon,
		ForecastWindow: forecastWindow,
	}

	// Validate API key before making request
	if strings.TrimSpace(w.APIKey) == "" {
		return forecast, errors.New("API key cannot be empty")
	}

	url := fmt.Sprintf("%s?lat=%f&lon=%f&units=metric&appid=%s", WeatherAPIBaseURL, lat, lon, w.APIKey)

	w.l.Info("making weatherapi API request", map[string]any{
		"params": forecast.RequestParams(),
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return forecast, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return forecast, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	w.l.Info("received weatherapi API response", map[string]any{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return forecast, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return forecast, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response WeatherAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return forecast, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	w.l.Info("parsed API response", map[string]any{
		"items": len(response.List),
	})

	// Check if we have any data
	if len(response.List) == 0 {
		return forecast, fmt.Errorf("no forecast data available")
	}

	// Process daily temperatures
	dailyTemps, err := dailyTemperaturesWeatherAPI(response)
	if err != nil {
		return forecast, fmt.Errorf("failed to process daily temperatures: %w", err)
	}

	forecast.ForecastData = dailyTemps[:forecastWindow]

	return forecast, nil
}

func dailyTemperaturesWeatherAPI(response WeatherAPIResponse) ([]models.WeatherData, error) {
	var dailyTemps []models.WeatherData

	// Group temperatures by date
	for _, item := range response.List {
		// Parse the date from dt_txt (format: "2025-07-25 18:00:00")
		date, err := parseDate(item.DtTxt)
		if err != nil {
			return dailyTemps, fmt.Errorf("failed to parse date from dt_txt %s: %w", item.DtTxt, err)
		}

		index := models.FilterByDate(dailyTemps, date)

		if index == -1 {
			// Create new entry for this date
			dailyTemps = append(dailyTemps, models.WeatherData{
				Date:    date,
				TempMin: item.Main.TempMin,
				TempMax: item.Main.TempMax,
			})
			continue
		}

		// Update existing entry
		if item.Main.TempMin < dailyTemps[index].TempMin {
			dailyTemps[index].TempMin = item.Main.TempMin
		}
		if item.Main.TempMax > dailyTemps[index].TempMax {
			dailyTemps[index].TempMax = item.Main.TempMax
		}
	}

	return dailyTemps, nil
}

func parseDate(dateStr string) (*time.Time, error) {
	if len(dateStr) < 10 {
		// Skip if the date format is unexpected
		return nil, fmt.Errorf("invalid date string: %s", dateStr)
	}

	dateStr = dateStr[:10] // Extract just the date part

	// Parse the date string in the format "2025-07-25"
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
	}

	return &t, nil
}
