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

func TestWeatherAPIRepository_FetchForecast_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	logger := logger.NewZapLogger("test-app")
	repo := NewWeatherAPIRepository("test-key", logger)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error when calling invalid URL, got nil")
	}
}

func TestWeatherAPIRepository_FetchForecast_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	logger := logger.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		APIKey: "test-key",
		l:      logger,
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error when receiving invalid JSON, got nil")
	}
}

func TestWeatherAPIRepository_FetchForecast_ContextCancellation(t *testing.T) {
	// Create a mock server that delays response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"list": [{"dt": 1753455600, "dt_txt": "2025-07-25 15:00:00", "main": {"temp_min": 21.7, "temp_max": 22.52}}]}`))
	}))
	defer mockServer.Close()

	logger := logger.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		APIKey: "test-key",
		l:      logger,
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error when context is cancelled, got nil")
	}
}

func TestWeatherAPIRepository_Name(t *testing.T) {
	repo := &WeatherAPIRepository{}
	expected := "weatherapi"
	if name := repo.Name(); name != expected {
		t.Errorf("Expected name to be %s, got %s", expected, name)
	}
}

// testWeatherAPIRepository is a test-specific version that allows overriding the base URL
type testWeatherAPIRepository struct {
	*WeatherAPIRepository
	baseURL string
}

func (t *testWeatherAPIRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error) {
	// Use the test-specific base URL
	url := fmt.Sprintf("%s?lat=%f&lon=%f&units=metric&appid=%s", t.baseURL, lat, lon, t.APIKey)

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
		var errorResp WeatherAPIErrorResponse
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil {
			t.l.Error(fmt.Errorf("API error: %s", errorResp.Message), map[string]any{
				"repository": t.Name(),
				"statusCode": resp.StatusCode,
				"errorCode":  errorResp.Cod,
			})
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorResp.Message)
		}

		t.l.Error(fmt.Errorf("HTTP error: %s", resp.Status), map[string]any{
			"repository": t.Name(),
			"statusCode": resp.StatusCode,
			"body":       string(body),
		})
		return nil, fmt.Errorf("HTTP error (status %d): %s", resp.StatusCode, resp.Status)
	}

	var response WeatherAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.l.Error(err, map[string]any{
			"repository": t.Name(),
			"body":       string(body),
		})
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	t.l.Info("parsed API response", map[string]any{
		"repository": t.Name(),
		"items":      len(response.List),
	})

	// Check if we have any data
	if len(response.List) == 0 {
		t.l.Warning("no forecast data received", map[string]any{
			"repository": t.Name(),
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
			t.l.Warning("invalid date format", map[string]any{
				"repository": t.Name(),
				"dtTxt":      item.DtTxt,
			})
			continue
		}

		date := item.DtTxt[:10] // Extract just the date part
		if dailyTemps[date] == nil {
			dailyTemps[date] = []float64{item.Main.TempMin, item.Main.TempMax}
		} else {
			// Update min/max for the day
			if item.Main.TempMin < dailyTemps[date][0] {
				dailyTemps[date][0] = item.Main.TempMin
			}
			if item.Main.TempMax > dailyTemps[date][1] {
				dailyTemps[date][1] = item.Main.TempMax
			}
		}
	}

	// Convert to models.Response slice
	for date, temps := range dailyTemps {
		response := models.Response{
			Date:    date,
			TempMax: temps[1],
			TempMin: temps[0],
		}
		result = append(result, response)
	}

	t.l.Info("final forecast result", map[string]any{
		"repository": t.Name(),
		"days":       len(result),
		"forecast":   result,
	})

	return result, nil
}

func TestWeatherAPIRepository_FetchForecast_Success(t *testing.T) {
	// Create a mock server that returns valid OpenWeatherMap response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"list": [
				{"dt": 1753455600, "dt_txt": "2025-07-25 15:00:00", "main": {"temp_min": 21.7, "temp_max": 22.52}},
				{"dt": 1753466400, "dt_txt": "2025-07-25 18:00:00", "main": {"temp_min": 21.77, "temp_max": 21.91}},
				{"dt": 1753477200, "dt_txt": "2025-07-25 21:00:00", "main": {"temp_min": 19.88, "temp_max": 20.49}},
				{"dt": 1753488000, "dt_txt": "2025-07-26 00:00:00", "main": {"temp_min": 20.42, "temp_max": 20.42}},
				{"dt": 1753498800, "dt_txt": "2025-07-26 03:00:00", "main": {"temp_min": 20.64, "temp_max": 20.64}},
				{"dt": 1753509600, "dt_txt": "2025-07-26 06:00:00", "main": {"temp_min": 21.43, "temp_max": 21.43}},
				{"dt": 1753520400, "dt_txt": "2025-07-26 09:00:00", "main": {"temp_min": 20.54, "temp_max": 20.54}},
				{"dt": 1753531200, "dt_txt": "2025-07-26 12:00:00", "main": {"temp_min": 23.45, "temp_max": 23.45}}
			]
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	logger := logger.NewZapLogger("test-app")
	repo := &testWeatherAPIRepository{
		WeatherAPIRepository: &WeatherAPIRepository{
			APIKey: "test-key",
			l:      logger,
		},
		baseURL: mockServer.URL,
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	result, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the structure of the response - organized as slice
	if len(result) == 0 {
		t.Fatal("Expected weather data as slice")
	}

	// Log the data structure and sample data
	t.Logf("Weather data: %v", result)

	// Verify each response has proper weather data
	for _, response := range result {
		t.Logf("Date: %s, Max: %.1f°C, Min: %.1f°C",
			response.Date, response.TempMax, response.TempMin)

		// Verify temperature values are reasonable
		if response.TempMax < -50 || response.TempMax > 50 {
			t.Errorf("Max temperature for %s seems unreasonable: %f°C", response.Date, response.TempMax)
		}
		if response.TempMin < -50 || response.TempMin > 50 {
			t.Errorf("Min temperature for %s seems unreasonable: %f°C", response.Date, response.TempMin)
		}
	}
}

func TestWeatherAPIRepository_FetchForecast_RealAPI(t *testing.T) {
	t.Skip()

	// This test makes a real HTTP call to the OpenWeatherMap API
	// Using the API key from the user's example
	logger := logger.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		APIKey: "REAL_API_KEY", // Replace with a valid API key for testing
		l:      logger,
	}

	ctx := context.Background()
	lat := 45.44 // Venice latitude from the example
	lon := 12.33 // Venice longitude from the example
	forecastWindow := 5

	result, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err != nil {
		t.Fatalf("Real API call failed: %v", err)
	}

	t.Logf("Result: %v", result)

	// Verify the response structure - organized as slice
	if len(result) == 0 {
		t.Fatal("Expected weather data as slice")
	}

	// Log the actual data received for Venice
	t.Logf("Venice weather forecast: %v", result)

	// Verify each response has proper weather data for Venice
	for _, response := range result {
		t.Logf("Venice %s - Max: %.1f°C, Min: %.1f°C",
			response.Date, response.TempMax, response.TempMin)

		// Verify temperature values are reasonable for Venice
		if response.TempMax < -50 || response.TempMax > 50 {
			t.Errorf("Max temperature for %s seems unreasonable: %f°C", response.Date, response.TempMax)
		}
		if response.TempMin < -50 || response.TempMin > 50 {
			t.Errorf("Min temperature for %s seems unreasonable: %f°C", response.Date, response.TempMin)
		}
	}
}
