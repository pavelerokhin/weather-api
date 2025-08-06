package repositories

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"weather-api/pkg/logger"
)

func TestWeatherAPIRepository_FetchForecast_Success(t *testing.T) {
	// Create mock HTTP client that returns valid response
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify the request URL contains expected parameters
			if !strings.Contains(req.URL.String(), "lat=40.7128") {
				t.Errorf("Expected latitude in URL, got: %s", req.URL.String())
			}
			if !strings.Contains(req.URL.String(), "lon=-74.0060") {
				t.Errorf("Expected longitude in URL, got: %s", req.URL.String())
			}
			if !strings.Contains(req.URL.String(), "appid=test-key") {
				t.Errorf("Expected API key in URL, got: %s", req.URL.String())
			}

			// Return mock response
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

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	result, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.ForecastData) == 0 {
		t.Fatal("Expected weather data, got empty result")
	}

	// Verify we got data for both days
	if len(result.ForecastData) != 2 {
		t.Errorf("Expected 2 days of weather data, got %d", len(result.ForecastData))
	}

	// Verify the first day (2025-07-25)
	expectedDate1, _ := time.Parse("2006-01-02", "2025-07-25")
	if result.ForecastData[0].Date == nil || !result.ForecastData[0].Date.Equal(expectedDate1) {
		t.Errorf("Expected date 2025-07-25, got %v", result.ForecastData[0].Date)
	}
	// The min temp should be 19.88 (lowest of all readings for that day)
	if result.ForecastData[0].TempMin != 19.88 {
		t.Errorf("Expected min temp 19.88, got %f", result.ForecastData[0].TempMin)
	}
	// The max temp should be 22.52 (highest of all readings for that day)
	if result.ForecastData[0].TempMax != 22.52 {
		t.Errorf("Expected max temp 22.52, got %f", result.ForecastData[0].TempMax)
	}

	// Verify the second day (2025-07-26)
	expectedDate2, _ := time.Parse("2006-01-02", "2025-07-26")
	if result.ForecastData[1].Date == nil || !result.ForecastData[1].Date.Equal(expectedDate2) {
		t.Errorf("Expected date 2025-07-26, got %v", result.ForecastData[1].Date)
	}
	// The min temp should be 20.42 (lowest of all readings for that day)
	if result.ForecastData[1].TempMin != 20.42 {
		t.Errorf("Expected min temp 20.42, got %f", result.ForecastData[1].TempMin)
	}
	// The max temp should be 23.45 (highest of all readings for that day)
	if result.ForecastData[1].TempMax != 23.45 {
		t.Errorf("Expected max temp 23.45, got %f", result.ForecastData[1].TempMax)
	}
}

func TestWeatherAPIRepository_FetchForecast_HTTPError(t *testing.T) {
	// Create mock HTTP client that returns HTTP error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"cod": "401", "message": "Invalid API key"}`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("invalid-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err = repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP error (status 401)") {
		t.Errorf("Expected HTTP error message, got: %v", err)
	}
}

func TestWeatherAPIRepository_FetchForecast_NetworkError(t *testing.T) {
	// Create mock HTTP client that returns network error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network connection failed")
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err = repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error for network failure, got nil")
	}
	if !strings.Contains(err.Error(), "network connection failed") {
		t.Errorf("Expected network error message, got: %v", err)
	}
}

func TestWeatherAPIRepository_FetchForecast_InvalidJSON(t *testing.T) {
	// Create mock HTTP client that returns invalid JSON
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("invalid json")),
				Header:     make(http.Header),
			}, nil
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err = repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON response") {
		t.Errorf("Expected JSON parsing error message, got: %v", err)
	}
}

func TestWeatherAPIRepository_FetchForecast_EmptyData(t *testing.T) {
	// Create mock HTTP client that returns empty data
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := `{"list": []}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err = repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err == nil {
		t.Error("Expected error for empty data, got nil")
	}
	if !strings.Contains(err.Error(), "no forecast data available") {
		t.Errorf("Expected no data error message, got: %v", err)
	}
}

func TestWeatherAPIRepository_FetchForecast_InvalidDateFormat(t *testing.T) {
	// Create mock HTTP client that returns data with invalid date format
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := `{
				"list": [
					{"dt": 1753455600, "dt_txt": "invalid-date", "main": {"temp_min": 21.7, "temp_max": 22.52}}
				]
			}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	result, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should filter out invalid date format
	if len(result.ForecastData) != 0 {
		t.Errorf("Expected 0 valid days, got %d", len(result.ForecastData))
	}
}

func TestWeatherAPIRepository_FetchForecast_ContextCancellation(t *testing.T) {
	// Create mock HTTP client that respects context cancellation
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Check if context is cancelled
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			default:
				// Simulate a delay
				time.Sleep(10 * time.Millisecond)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"list": [{"dt": 1753455600, "dt_txt": "2025-05-25 15:00:00", "main": {"temp_min": 21.7, "temp_max": 22.52}}]}`)),
					Header:     make(http.Header),
				}, nil
			}
		},
	}

	l := logger.NewZapLogger("test-app")
	repo, err := NewWeatherAPIRepository("test-key", l, mockClient)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 40.7128
	lon := -74.0060
	forecastWindow := 5

	_, err = repo.FetchForecast(ctx, lat, lon, forecastWindow)
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

func TestWeatherAPIRepository_RealAPI(t *testing.T) {
	t.Skip("Skipping real API test - uncomment to test against actual OpenWeatherMap API")

	// This test makes a real HTTP call to the OpenWeatherMap API
	l := logger.NewZapLogger("test-app")
	httpClient := &DefaultHTTPClient{}
	repo, err := NewWeatherAPIRepository("REAL_API_KEY", l, httpClient) // Replace with valid API key

	ctx := context.Background()
	lat := 45.44 // Venice latitude
	lon := 12.33 // Venice longitude
	forecastWindow := 5

	result, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
	if err != nil {
		t.Fatalf("Real API call failed: %v", err)
	}

	if len(result.ForecastData) == 0 {
		t.Fatal("Expected weather data, got empty result")
	}

	// Verify each response has proper weather data
	for _, response := range result.ForecastData {
		// Verify temperature values are reasonable for Venice
		if response.TempMax < -50 || response.TempMax > 50 {
			t.Errorf("Max temperature for %s seems unreasonable: %f°C", response.Date, response.TempMax)
		}
		if response.TempMin < -50 || response.TempMin > 50 {
			t.Errorf("Min temperature for %s seems unreasonable: %f°C", response.Date, response.TempMin)
		}
	}
}
