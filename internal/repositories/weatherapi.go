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
	WeatherAPIBaseURL = "https://api.openweathermap.org/data/2.5/forecast"
)

type WeatherAPIRepository struct {
	APIKey string
	l      *observe.Logger
}

func NewWeatherAPIRepository(apiKey string, l *observe.Logger) *WeatherAPIRepository {
	return &WeatherAPIRepository{
		APIKey: apiKey,
		l:      l,
	}
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

type WeatherAPIErrorResponse struct {
	Cod     string `json:"cod"`
	Message string `json:"message"`
}

func (w *WeatherAPIRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error) {
	url := fmt.Sprintf("%s?lat=%f&lon=%f&units=metric&appid=%s", OpenMeteoBaseURL, lat, lon, w.APIKey)

	w.l.Info("making API request", map[string]any{
		"repository": w.Name(),
		"url":        url,
		"lat":        lat,
		"lon":        lon,
		"window":     forecastWindow,
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		w.l.Error(err, map[string]any{
			"repository": w.Name(),
		})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.l.Error(err, map[string]any{
			"repository": w.Name(),
		})
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	w.l.Info("received API response", map[string]any{
		"repository": w.Name(),
		"status":     resp.StatusCode,
		"statusText": resp.Status,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.l.Error(err, map[string]any{
			"repository": w.Name(),
		})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP error status codes
	if resp.StatusCode != http.StatusOK {
		var errorResp WeatherAPIErrorResponse
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil {
			w.l.Error(fmt.Errorf("API error: %s", errorResp.Message), map[string]any{
				"repository": w.Name(),
				"statusCode": resp.StatusCode,
				"errorCode":  errorResp.Cod,
			})
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorResp.Message)
		}

		w.l.Error(fmt.Errorf("HTTP error: %s", resp.Status), map[string]any{
			"repository": w.Name(),
			"statusCode": resp.StatusCode,
			"body":       string(body),
		})
		return nil, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response WeatherAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		w.l.Error(err, map[string]any{
			"repository": w.Name(),
			"body":       string(body),
		})
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	w.l.Info("parsed API response", map[string]any{
		"repository": w.Name(),
		"items":      len(response.List),
	})

	// Check if we have any data
	if len(response.List) == 0 {
		w.l.Warning("no forecast data received", map[string]any{
			"repository": w.Name(),
		})
		return nil, fmt.Errorf("no forecast data available")
	}

	// Organize data by date and convert to models.Response slice
	var result []models.Response
	dailyTemps := make(map[string][]float64) // date -> []{min, max}

	// Group temperatures by date
	for _, item := range response.List {
		// Parse the date from dt_txt (format: "2025-07-25 18:00:00")
		if len(item.DtTxt) < 10 {
			w.l.Warning("invalid date format", map[string]any{
				"repository": w.Name(),
				"dtTxt":      item.DtTxt,
			})
			continue
		}

		date := item.DtTxt[:10] // Extract just the date part

		if temps, exists := dailyTemps[date]; exists {
			// Update min/max for existing date
			if item.Main.TempMin < temps[0] {
				temps[0] = item.Main.TempMin
			}
			if item.Main.TempMax > temps[1] {
				temps[1] = item.Main.TempMax
			}
			dailyTemps[date] = temps
		} else {
			// Initialize min/max for new date
			dailyTemps[date] = []float64{item.Main.TempMin, item.Main.TempMax}
		}
	}

	w.l.Info("processed daily temperatures", map[string]any{
		"repository": w.Name(),
		"dates":      len(dailyTemps),
		"dailyTemps": dailyTemps,
	})

	// Convert to final result format and limit to forecastWindow days
	count := 0
	for date, temps := range dailyTemps {
		if count >= forecastWindow {
			break
		}
		response := models.Response{
			Date:    date,
			TempMax: temps[1],
			TempMin: temps[0],
		}
		result = append(result, response)
		count++
	}

	w.l.Info("final forecast result", map[string]any{
		"repository": w.Name(),
		"days":       len(result),
		"forecast":   result,
	})

	return result, nil
}
