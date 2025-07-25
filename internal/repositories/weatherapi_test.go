package repositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"weather-api/pkg/observe"
)

func TestWeatherAPIRepository_FetchForecast_ErrorHandling(t *testing.T) {
	// Test with invalid URL
	logger := observe.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		BaseURL: "http://invalid-url-that-does-not-exist.com",
		APIKey:  "test-key",
		l:       logger,
	}

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

	logger := observe.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		BaseURL: mockServer.URL,
		APIKey:  "test-key",
		l:       logger,
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

	logger := observe.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		BaseURL: mockServer.URL,
		APIKey:  "test-key",
		l:       logger,
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

	logger := observe.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		BaseURL: mockServer.URL,
		APIKey:  "test-key",
		l:       logger,
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
	// This test makes a real HTTP call to the OpenWeatherMap API
	// Using the API key from the user's example
	logger := observe.NewZapLogger("test-app")
	repo := &WeatherAPIRepository{
		BaseURL: "https://api.openweathermap.org/data/2.5/forecast",
		APIKey:  "983363b3c2a5b9560d9170f8375eca2e",
		l:       logger,
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
