package repositories

import (
	"context"

	"weather-api/config"
	"weather-api/internal/models"
	"weather-api/pkg/logger"
)

type WeatherRepository interface {
	Name() string
	FetchForecast(ctx context.Context, lat, lon float64, forecastWindow int) ([]models.Response, error)
}

func InitWeatherRepositories(cfg *config.Config, l *logger.Logger) []WeatherRepository {
	var repos []WeatherRepository
	for _, api := range cfg.Weather.APIs {
		switch api.Name {
		case "open-meteo":
			repos = append(repos, &OpenMeteoRepository{
				l: l,
			})
		case "weatherapi":
			repos = append(repos, &WeatherAPIRepository{
				APIKey: api.APIKey,
				l:      l,
			})
			// add more cases for new providers to extend the app
		}
	}

	return repos
}
