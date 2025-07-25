package repositories

import (
	"context"
	"weather-api/pkg/observe"

	"weather-api/config"
	"weather-api/internal/models"
)

type WeatherRepository interface {
	Name() string
	FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error)
}

func InitWeatherRepositories(cfg *config.Config, l *observe.Logger) []WeatherRepository {
	var repos []WeatherRepository
	for _, api := range cfg.WeatherAPIs {
		switch api.Name {
		case "open-meteo":
			repos = append(repos, &OpenMeteoRepository{
				BaseURL: api.BaseURL,
				l:       l,
			})
		case "weatherapi":
			repos = append(repos, &WeatherAPIRepository{
				BaseURL: api.BaseURL,
				APIKey:  api.APIKey,
				l:       l,
			})
			// Add more cases for new providers toi extyend the app
		}
	}

	return repos
}
