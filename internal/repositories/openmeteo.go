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

type OpenMeteoRepository struct {
	BaseURL string
	l       *observe.Logger
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
	// Always fetch a 5-day forecast
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min&forecast_days=%d&timezone=auto", o.BaseURL, lat, lon, forecastWindow)

	o.l.Info("making API request", map[string]any{
		"repository": o.Name(),
		"url":        url,
		"lat":        lat,
		"lon":        lon,
		"window":     forecastWindow,
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		o.l.Error(err, map[string]any{
			"repository": o.Name(),
		})
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

	// Check if we have any data
	if len(response.Daily.Time) == 0 {
		o.l.Warning("no forecast data received", map[string]any{
			"repository": o.Name(),
		})
		return nil, fmt.Errorf("no forecast data available")
	}

	// Convert to models.Response slice
	var result []models.Response

	// Find the minimum length to avoid index out of bounds
	minLength := len(response.Daily.Time)
	if len(response.Daily.Temperature2mMax) < minLength {
		minLength = len(response.Daily.Temperature2mMax)
	}
	if len(response.Daily.Temperature2mMin) < minLength {
		minLength = len(response.Daily.Temperature2mMin)
	}

	// Check if we have enough data
	if minLength == 0 {
		o.l.Warning("insufficient temperature data", map[string]any{
			"repository": o.Name(),
			"timeLength": len(response.Daily.Time),
			"maxLength":  len(response.Daily.Temperature2mMax),
			"minLength":  len(response.Daily.Temperature2mMin),
		})
		return nil, fmt.Errorf("insufficient temperature data available")
	}

	for i := 0; i < minLength; i++ {
		// Validate temperature data
		if response.Daily.Temperature2mMax[i] < response.Daily.Temperature2mMin[i] {
			o.l.Warning("invalid temperature data (max < min)", map[string]any{
				"repository": o.Name(),
				"date":       response.Daily.Time[i],
				"max":        response.Daily.Temperature2mMax[i],
				"min":        response.Daily.Temperature2mMin[i],
			})
			continue
		}

		response := models.Response{
			Date:    response.Daily.Time[i],
			TempMax: response.Daily.Temperature2mMax[i],
			TempMin: response.Daily.Temperature2mMin[i],
		}
		result = append(result, response)
	}

	o.l.Info("final forecast result", map[string]any{
		"repository": o.Name(),
		"days":       len(result),
		"forecast":   result,
	})

	return result, nil
}
