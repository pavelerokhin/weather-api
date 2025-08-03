package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"weather-api/internal/models"
	"weather-api/pkg/logger"
)

// testOpenMeteoRepository is a test-specific version that allows overriding the base URL
type testOpenMeteoRepository struct {
	*OpenMeteoRepository
	baseURL string
}

func (t *testOpenMeteoRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error) {
	// Use the test-specific base URL
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min&forecast_days=%d&timezone=auto", t.baseURL, lat, lon, forecastWindow)

	t.l.Info("making API request", map[string]any{
		"repository": t.Name(),
		"url":        url,
		"lat":        lat,
		"lon":        lon,
		"window":     forecastWindow,
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.l.Error(err, map[string]any{
			"repository": t.Name(),
		})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.l.Error(err, map[string]any{
			"repository": t.Name(),
		})
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	t.l.Info("received API response", map[string]any{
		"repository": t.Name(),
		"status":     resp.StatusCode,
		"statusText": resp.Status,
	})

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.l.Error(err, map[string]any{
			"repository": t.Name(),
		})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP error status codes
	if resp.StatusCode != http.StatusOK {
		var errorResp OpenMeteoErrorResponse
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil && errorResp.Error {
			t.l.Error(fmt.Errorf("API error: %s", errorResp.Reason), map[string]any{
				"repository": t.Name(),
				"statusCode": resp.StatusCode,
				"reason":     errorResp.Reason,
				"message":    errorResp.Message,
			})
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorResp.Reason)
		}

		t.l.Error(fmt.Errorf("HTTP error: %s", resp.Status), map[string]any{
			"repository": t.Name(),
			"statusCode": resp.StatusCode,
			"body":       string(body),
		})
		return nil, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		Daily OpenMeteoResponse `json:"daily"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.l.Error(err, map[string]any{
			"repository": t.Name(),
			"body":       string(body),
		})
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	t.l.Info("parsed API response", map[string]any{
		"repository": t.Name(),
		"days":       len(response.Daily.Time),
	})

	// Validate that we have forecast data
	if !t.hasForecastData(response.Daily) {
		t.l.Warning("no forecast data received", map[string]any{
			"repository": t.Name(),
		})
		return nil, fmt.Errorf("no forecast data available")
	}

	// Convert API response to weather forecast models
	forecastDays := t.buildForecastFromResponse(response.Daily)

	t.l.Info("final forecast result", map[string]any{
		"repository": t.Name(),
		"days":       len(forecastDays),
		"forecast":   forecastDays,
	})

	return forecastDays, nil
}

// hasForecastData checks if the daily forecast data contains valid time entries
func (t *testOpenMeteoRepository) hasForecastData(daily OpenMeteoResponse) bool {
	return len(daily.Time) > 0
}

// buildForecastFromResponse converts the API response to weather forecast models
func (t *testOpenMeteoRepository) buildForecastFromResponse(daily OpenMeteoResponse) []models.Response {
	var forecastDays []models.Response

	// Find the minimum length to avoid index out of bounds
	minLength := min(len(daily.Time), len(daily.Temperature2mMax), len(daily.Temperature2mMin))

	// Check if we have enough data
	if minLength == 0 {
		t.l.Warning("insufficient temperature data", map[string]any{
			"repository": t.Name(),
			"timeLength": len(daily.Time),
			"maxLength":  len(daily.Temperature2mMax),
			"minLength":  len(daily.Temperature2mMin),
		})
		return nil
	}

	// Build forecast for each day
	for i := 0; i < minLength; i++ {
		dayForecast := t.createDayForecast(daily, i)
		if dayForecast != nil {
			forecastDays = append(forecastDays, *dayForecast)
		}
	}

	return forecastDays
}

// createDayForecast creates a single day forecast, validating temperature data
func (t *testOpenMeteoRepository) createDayForecast(daily OpenMeteoResponse, index int) *models.Response {
	maxTemp := daily.Temperature2mMax[index]
	minTemp := daily.Temperature2mMin[index]

	// Validate temperature data (max should be >= min)
	if maxTemp < minTemp {
		t.l.Warning("invalid temperature data (max < min)", map[string]any{
			"repository": t.Name(),
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

func TestOpenMeteoRepository_FetchForecast_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	logger := logger.NewZapLogger("test-app")

	repo := &testOpenMeteoRepository{
		OpenMeteoRepository: &OpenMeteoRepository{l: logger},
		baseURL:             "http://invalid-url-that-does-not-exist.com",
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060

	_, err := repo.FetchForecast(ctx, lat, lon, 5)
	if err == nil {
		t.Error("Expected error when calling invalid URL, got nil")
	}
}

func TestOpenMeteoRepository_FetchForecast_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	logger := logger.NewZapLogger("test-app")

	repo := &testOpenMeteoRepository{
		OpenMeteoRepository: &OpenMeteoRepository{l: logger},
		baseURL:             mockServer.URL,
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060

	_, err := repo.FetchForecast(ctx, lat, lon, 5)
	if err == nil {
		t.Error("Expected error when receiving invalid JSON, got nil")
	}
}

func TestOpenMeteoRepository_FetchForecast_ContextCancellation(t *testing.T) {
	// Create a mock server that delays response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"daily": {"temperature_2m_max": [25.5], "temperature_2m_min": [15.2], "precipitation_sum": [0.0]}}`))
	}))
	defer mockServer.Close()

	logger := logger.NewZapLogger("test-app")
	repo := &testOpenMeteoRepository{
		OpenMeteoRepository: &OpenMeteoRepository{l: logger},
		baseURL:             mockServer.URL,
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 40.7128
	lon := -74.0060

	_, err := repo.FetchForecast(ctx, lat, lon, 1)
	if err == nil {
		t.Error("Expected error when context is cancelled, got nil")
	}
}

func TestOpenMeteoRepository_Name(t *testing.T) {
	repo := &OpenMeteoRepository{}
	expected := "open-meteo"
	if name := repo.Name(); name != expected {
		t.Errorf("Expected name to be %s, got %s", expected, name)
	}
}

func TestOpenMeteoRepository_FetchForecast_Success(t *testing.T) {
	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	result, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Expected weather data as slice")
	}

	expectedDays := 2
	if len(result) != expectedDays {
		t.Errorf("Expected %d days of weather data, got %d", expectedDays, len(result))
	}

	t.Logf("Weather data: %v", result)

	for _, response := range result {
		t.Logf("Date: %s, Max: %.1f°C, Min: %.1f°C",
			response.Date, response.TempMax, response.TempMin)
	}
}

func TestOpenMeteoRepository_FetchForecast_RealAPI(t *testing.T) {
	// This test makes a real HTTP call to the Open-Meteo API
	// with the exact parameters from the user's request
	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger)

	ctx := context.Background()
	lat := 52.52 // Berlin latitude
	lon := 13.41 // Berlin longitude

	result, err := repo.FetchForecast(ctx, lat, lon, 3)
	if err != nil {
		t.Fatalf("Real API call failed: %v", err)
	}

	t.Logf("Result: %v", result)

	// Verify the response structure - organized as slice
	if len(result) == 0 {
		t.Fatal("Expected weather data as slice")
	}

	// Log the actual data received for Berlin
	t.Logf("Berlin weather forecast: %v", result)

	// Verify each response has proper weather data for Berlin
	for _, response := range result {
		t.Logf("Berlin %s - Max: %.1f°C, Min: %.1f°C",
			response.Date, response.TempMax, response.TempMin)

		// Verify temperature values are reasonable for Berlin
		if response.TempMax < -50 || response.TempMax > 50 {
			t.Errorf("Max temperature for %s seems unreasonable: %f°C", response.Date, response.TempMax)
		}
		if response.TempMin < -50 || response.TempMin > 50 {
			t.Errorf("Min temperature for %s seems unreasonable: %f°C", response.Date, response.TempMin)
		}
	}
}
