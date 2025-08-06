package weather_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weather-api/internal/models"
	"weather-api/internal/repositories"
	"weather-api/internal/services/weather"
	"weather-api/pkg/logger"
)

// MockRepository implements WeatherRepository for testing
type MockRepository struct {
	name         string
	shouldFail   bool
	shouldDelay  bool
	forecastData models.Forecast
	callCount    int
}

func (m *MockRepository) Name() string {
	return m.name
}

func (m *MockRepository) FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) (models.Forecast, error) {
	m.callCount++

	if m.shouldDelay {
		select {
		case <-ctx.Done():
			return models.Forecast{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	if m.shouldFail {
		return models.Forecast{}, errors.New("mock repository error")
	}

	return m.forecastData, nil
}

func TestNewWeatherService(t *testing.T) {
	l := logger.NewZapLogger("test-app")
	repos := []repositories.WeatherRepository{
		&MockRepository{name: "test-repo-1"},
		&MockRepository{name: "test-repo-2"},
	}

	service := weather.NewWeatherService(repos, l)

	assert.NotNil(t, service)
}

func TestWeatherService_FetchForecasts_Success(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	date1 := time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 7, 26, 0, 0, 0, 0, time.UTC)

	mockForecast1 := models.Forecast{
		RepositoryName: "repo-1",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 2,
		ForecastData: []models.WeatherData{
			{Date: &date1, TempMax: 25.0, TempMin: 15.0},
			{Date: &date2, TempMax: 26.0, TempMin: 16.0},
		},
	}

	mockForecast2 := models.Forecast{
		RepositoryName: "repo-2",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 2,
		ForecastData: []models.WeatherData{
			{Date: &date1, TempMax: 24.5, TempMin: 14.5},
			{Date: &date2, TempMax: 25.5, TempMin: 15.5},
		},
	}

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "repo-1", forecastData: mockForecast1},
		&MockRepository{name: "repo-2", forecastData: mockForecast2},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	assert.Equal(t, mockForecast1, results["repo-1"])
	assert.Equal(t, mockForecast2, results["repo-2"])
}

func TestWeatherService_FetchForecasts_PartialFailure(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	date1 := time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 7, 26, 0, 0, 0, 0, time.UTC)

	mockForecast := models.Forecast{
		RepositoryName: "success-repo",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 2,
		ForecastData: []models.WeatherData{
			{Date: &date1, TempMax: 25.0, TempMin: 15.0},
			{Date: &date2, TempMax: 26.0, TempMin: 16.0},
		},
	}

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "success-repo", forecastData: mockForecast},
		&MockRepository{name: "failure-repo", shouldFail: true},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2) // Both repos should be in results

	assert.Equal(t, mockForecast, results["success-repo"])
	assert.Equal(t, "failure-repo", results["failure-repo"].RepositoryName)
	assert.Empty(t, results["failure-repo"].ForecastData)
}

func TestWeatherService_FetchForecasts_AllFailures(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "failure-repo-1", shouldFail: true},
		&MockRepository{name: "failure-repo-2", shouldFail: true},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2) // Both failed repos should be included with empty arrays
	assert.Equal(t, "failure-repo-1", results["failure-repo-1"].RepositoryName)
	assert.Empty(t, results["failure-repo-1"].ForecastData)
	assert.Equal(t, "failure-repo-2", results["failure-repo-2"].RepositoryName)
	assert.Empty(t, results["failure-repo-2"].ForecastData)
}

func TestWeatherService_FetchForecasts_EmptyRepositories(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	repos := []repositories.WeatherRepository{}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0) // Empty map when no repositories
}

func TestWeatherService_FetchForecasts_ContextCancellation(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "delayed-repo", shouldDelay: true},
	}

	service := weather.NewWeatherService(repos, l)

	ctx, cancel := context.WithCancel(context.Background())
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 2

	// Cancel context immediately
	cancel()

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1) // Failed repo should be included with empty array
	assert.Equal(t, "delayed-repo", results["delayed-repo"].RepositoryName)
	assert.Empty(t, results["delayed-repo"].ForecastData)
}

func TestWeatherService_FetchForecasts_ConcurrentExecution(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	date1 := time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC)

	// Create multiple repositories with different delays to test concurrency
	repos := []repositories.WeatherRepository{
		&MockRepository{
			name: "fast-repo",
			forecastData: models.Forecast{
				RepositoryName: "fast-repo",
				Lat:            40.7128,
				Lon:            -74.0060,
				ForecastWindow: 1,
				ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 25.0, TempMin: 15.0}},
			},
		},
		&MockRepository{
			name: "medium-repo",
			forecastData: models.Forecast{
				RepositoryName: "medium-repo",
				Lat:            40.7128,
				Lon:            -74.0060,
				ForecastWindow: 1,
				ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 24.0, TempMin: 14.0}},
			},
		},
		&MockRepository{
			name: "slow-repo",
			forecastData: models.Forecast{
				RepositoryName: "slow-repo",
				Lat:            40.7128,
				Lon:            -74.0060,
				ForecastWindow: 1,
				ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 23.0, TempMin: 13.0}},
			},
		},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 1

	start := time.Now()
	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 3)

	// Verify all repositories were called
	assert.Equal(t, 1, repos[0].(*MockRepository).callCount)
	assert.Equal(t, 1, repos[1].(*MockRepository).callCount)
	assert.Equal(t, 1, repos[2].(*MockRepository).callCount)

	// Verify results contain all repositories
	assert.Contains(t, results, "fast-repo")
	assert.Contains(t, results, "medium-repo")
	assert.Contains(t, results, "slow-repo")

	// The execution should be concurrent, so it should complete quickly
	// (much faster than if it were sequential)
	assert.Less(t, duration, 200*time.Millisecond)
}

func TestWeatherService_FetchForecasts_DefaultForecastWindow(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	date1 := time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC)

	mockForecast := models.Forecast{
		RepositoryName: "test-repo",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 0,
		ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 25.0, TempMin: 15.0}},
	}

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "test-repo", forecastData: mockForecast},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 0 // Use default

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, mockForecast, results["test-repo"])
}

func TestWeatherService_FetchForecasts_InvalidCoordinates(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "test-repo", shouldFail: true}, // Will fail with invalid coordinates
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 999.0 // Invalid latitude
	lon := 999.0 // Invalid longitude
	forecastWindow := 2

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1) // Failed repo should be included with empty array
	assert.Equal(t, "test-repo", results["test-repo"].RepositoryName)
	assert.Empty(t, results["test-repo"].ForecastData)
}

func TestWeatherService_FetchForecasts_MixedSuccessAndFailure(t *testing.T) {
	l := logger.NewZapLogger("test-app")

	date1 := time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC)

	mockForecast1 := models.Forecast{
		RepositoryName: "success-1",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 1,
		ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 25.0, TempMin: 15.0}},
	}

	mockForecast2 := models.Forecast{
		RepositoryName: "success-2",
		Lat:            40.7128,
		Lon:            -74.0060,
		ForecastWindow: 1,
		ForecastData:   []models.WeatherData{{Date: &date1, TempMax: 24.0, TempMin: 14.0}},
	}

	repos := []repositories.WeatherRepository{
		&MockRepository{name: "success-1", forecastData: mockForecast1},
		&MockRepository{name: "failure-1", shouldFail: true},
		&MockRepository{name: "success-2", forecastData: mockForecast2},
		&MockRepository{name: "failure-2", shouldFail: true},
	}

	service := weather.NewWeatherService(repos, l)

	ctx := context.Background()
	lat := 40.7128
	lon := -74.0060
	forecastWindow := 1

	results, err := service.FetchForecasts(ctx, lat, lon, forecastWindow)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 4) // All repos should be in results

	assert.Equal(t, mockForecast1, results["success-1"])
	assert.Equal(t, mockForecast2, results["success-2"])
	assert.Equal(t, "failure-1", results["failure-1"].RepositoryName)
	assert.Empty(t, results["failure-1"].ForecastData)
	assert.Equal(t, "failure-2", results["failure-2"].RepositoryName)
	assert.Empty(t, results["failure-2"].ForecastData)
}
