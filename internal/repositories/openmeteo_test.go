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

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, fmt.Errorf("mock not implemented")
}

func TestOpenMeteoRepository_FetchForecast_Success(t *testing.T) {
	// Create mock HTTP client that returns valid response
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify the request URL contains expected parameters
			if !strings.Contains(req.URL.String(), "latitude=52.52") {
				t.Errorf("Expected latitude in URL, got: %s", req.URL.String())
			}
			if !strings.Contains(req.URL.String(), "longitude=13.41") {
				t.Errorf("Expected longitude in URL, got: %s", req.URL.String())
			}
			if !strings.Contains(req.URL.String(), "forecast_days=2") {
				t.Errorf("Expected forecast_days in URL, got: %s", req.URL.String())
			}

			// Return mock response
			response := `{
				"daily": {
					"time": ["2025-01-27", "2025-01-28"],
					"temperature_2m_max": [25.5, 26.2],
					"temperature_2m_min": [15.2, 16.1]
				}
			}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	result, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.ForecastData) != 2 {
		t.Errorf("Expected 2 days of weather data, got %d", len(result.ForecastData))
	}

	// Verify the first day
	expectedDate1, _ := time.Parse("2006-01-02", "2025-01-27")
	if result.ForecastData[0].Date == nil || !result.ForecastData[0].Date.Equal(expectedDate1) {
		t.Errorf("Expected date 2025-01-27, got %v", result.ForecastData[0].Date)
	}
	if result.ForecastData[0].TempMax != 25.5 {
		t.Errorf("Expected max temp 25.5, got %f", result.ForecastData[0].TempMax)
	}
	if result.ForecastData[0].TempMin != 15.2 {
		t.Errorf("Expected min temp 15.2, got %f", result.ForecastData[0].TempMin)
	}

	// Verify the second day
	expectedDate2, _ := time.Parse("2006-01-02", "2025-01-28")
	if result.ForecastData[1].Date == nil || !result.ForecastData[1].Date.Equal(expectedDate2) {
		t.Errorf("Expected date 2025-01-28, got %v", result.ForecastData[1].Date)
	}
	if result.ForecastData[1].TempMax != 26.2 {
		t.Errorf("Expected max temp 26.2, got %f", result.ForecastData[1].TempMax)
	}
	if result.ForecastData[1].TempMin != 16.1 {
		t.Errorf("Expected min temp 16.1, got %f", result.ForecastData[1].TempMin)
	}
}

func TestOpenMeteoRepository_FetchForecast_HTTPError(t *testing.T) {
	// Create mock HTTP client that returns HTTP error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
				Header:     make(http.Header),
			}, nil
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	_, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP error (status 500)") {
		t.Errorf("Expected HTTP error message, got: %v", err)
	}
}

func TestOpenMeteoRepository_FetchForecast_NetworkError(t *testing.T) {
	// Create mock HTTP client that returns network error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network connection failed")
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	_, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err == nil {
		t.Error("Expected error for network failure, got nil")
	}
	if !strings.Contains(err.Error(), "network connection failed") {
		t.Errorf("Expected network error message, got: %v", err)
	}
}

func TestOpenMeteoRepository_FetchForecast_InvalidJSON(t *testing.T) {
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

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	_, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON response") {
		t.Errorf("Expected JSON parsing error message, got: %v", err)
	}
}

func TestOpenMeteoRepository_FetchForecast_EmptyData(t *testing.T) {
	// Create mock HTTP client that returns empty data
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := `{
				"daily": {
					"time": [],
					"temperature_2m_max": [],
					"temperature_2m_min": []
				}
			}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	_, err := repo.FetchForecast(ctx, lat, lon, 2)
	if err == nil {
		t.Error("Expected error for empty data, got nil")
	}
	if !strings.Contains(err.Error(), "no forecast data available") {
		t.Errorf("Expected no data error message, got: %v", err)
	}
}

func TestOpenMeteoRepository_FetchForecast_InvalidTemperatureData(t *testing.T) {
	// Create mock HTTP client that returns invalid temperature data (max < min)
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			response := `{
				"daily": {
					"time": ["2025-01-27"],
					"temperature_2m_max": [15.0],
					"temperature_2m_min": [20.0]
				}
			}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response)),
				Header:     make(http.Header),
			}, nil
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	ctx := context.Background()
	lat := 52.52
	lon := 13.41

	result, err := repo.FetchForecast(ctx, lat, lon, 1)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should filter out invalid temperature data
	if len(result.ForecastData) != 0 {
		t.Errorf("Expected 0 valid days, got %d", len(result.ForecastData))
	}
}

func TestOpenMeteoRepository_FetchForecast_ContextCancellation(t *testing.T) {
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
					Body:       io.NopCloser(strings.NewReader(`{"daily": {"time": ["2025-01-27"], "temperature_2m_max": [25.5], "temperature_2m_min": [15.2]}}`)),
					Header:     make(http.Header),
				}, nil
			}
		},
	}

	logger := logger.NewZapLogger("test-app")
	repo := NewOpenMeteoRepository(logger, mockClient)

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 52.52
	lon := 13.41

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

func TestOpenMeteoRepository_RealAPI(t *testing.T) {
	t.Skip("Skipping real API test - uncomment to test against actual Open-Meteo API")

	// This test makes a real HTTP call to the Open-Meteo API
	logger := logger.NewZapLogger("test-app")
	httpClient := &DefaultHTTPClient{}
	repo := NewOpenMeteoRepository(logger, httpClient)

	ctx := context.Background()
	lat := 52.52 // Berlin latitude
	lon := 13.41 // Berlin longitude

	result, err := repo.FetchForecast(ctx, lat, lon, 3)
	if err != nil {
		t.Fatalf("Real API call failed: %v", err)
	}

	if len(result.ForecastData) == 0 {
		t.Fatal("Expected weather data, got empty result")
	}

	// Verify each response has proper weather data
	for _, response := range result.ForecastData {
		// Verify temperature values are reasonable for Berlin
		if response.TempMax < -50 || response.TempMax > 50 {
			t.Errorf("Max temperature for %s seems unreasonable: %f°C", response.Date, response.TempMax)
		}
		if response.TempMin < -50 || response.TempMin > 50 {
			t.Errorf("Min temperature for %s seems unreasonable: %f°C", response.Date, response.TempMin)
		}
	}
}
