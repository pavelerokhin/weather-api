package repositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"weather-api/pkg/observe"
)

func TestOpenMeteoRepository_FetchForecast_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	logger := observe.NewZapLogger("test-app")
	repo := &OpenMeteoRepository{
		BaseURL: "http://invalid-url-that-does-not-exist.com",
		l:       logger,
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

	logger := observe.NewZapLogger("test-app")
	repo := &OpenMeteoRepository{
		BaseURL: mockServer.URL,
		l:       logger,
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

	logger := observe.NewZapLogger("test-app")
	repo := &OpenMeteoRepository{
		BaseURL: mockServer.URL,
		l:       logger,
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
	logger := observe.NewZapLogger("test-app")
	repo := &OpenMeteoRepository{
		BaseURL: "https://api.open-meteo.com/v1/forecast",
		l:       logger,
	}

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
	logger := observe.NewZapLogger("test-app")
	repo := &OpenMeteoRepository{
		BaseURL: "https://api.open-meteo.com/v1/forecast",
		l:       logger,
	}

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
