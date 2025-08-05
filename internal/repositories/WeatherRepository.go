package repositories

import (
	"context"
	"net/http"

	"weather-api/config"
	"weather-api/internal/models"
	"weather-api/pkg/logger"
)

// HTTPClient interface for making HTTP requests
// This allows for easy mocking in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient wraps the standard http.DefaultClient
type DefaultHTTPClient struct{}

func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

type WeatherRepository interface {
	Name() string
	FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error)
}

func InitWeatherRepositories(cfg *config.Config, l *logger.Logger) []WeatherRepository {
	var repos []WeatherRepository
	httpClient := &DefaultHTTPClient{}

	for _, api := range cfg.Weather.APIs {
		switch api.Name {
		case "open-meteo":
			repos = append(repos, NewOpenMeteoRepository(l, httpClient))
		case "weatherapi":
			repos = append(repos, NewWeatherAPIRepository(api.APIKey, l, httpClient))
			// add more cases for new providers to extend the app
		}
	}

	return repos
}
