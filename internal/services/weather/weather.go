package weather

import (
	"context"
	"sync"

	"weather-api/internal/models"
	"weather-api/internal/repositories"
	"weather-api/pkg/logger"
)

// WeatherService represents the weather service.
type WeatherService struct {
	repos []repositories.WeatherRepository
	l     *logger.Logger
}

func NewWeatherService(repos []repositories.WeatherRepository, l *logger.Logger) *WeatherService {
	return &WeatherService{
		repos: repos,
		l:     l,
	}
}

// FetchForecasts fetches the weather forecasts from all available APIs for the given latitude and longitude
func (s *WeatherService) FetchForecasts(ctx context.Context, lat, lon float64, forecastWindow int) (map[string]models.Forecast, error) {
	s.l.Info("starting forecast fetch", map[string]any{
		"lat":            lat,
		"lon":            lon,
		"forecastWindow": forecastWindow,
		"repositories":   len(s.repos),
	})

	results := make(map[string]models.Forecast)
	resultsChan := make(chan models.Forecast)
	var wg sync.WaitGroup

	for _, repo := range s.repos {
		wg.Add(1)
		go func(repo repositories.WeatherRepository) {
			defer wg.Done()
			s.l.Debug("fetching forecast", map[string]any{"repo": repo.Name(), "lat": lat, "lon": lon})

			forecast, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
			if err != nil {
				s.l.Error(err, map[string]any{"repo": repo.Name(), "err": err})

				resultsChan <- models.Forecast{
					RepositoryName: repo.Name(),
					Lat:            lat,
					Lon:            lon,
					ForecastWindow: forecastWindow,
					ForecastData:   []models.WeatherData{},
				}

				return
			}

			s.l.Info("successfully fetched forecast", map[string]any{
				"repo": repo.Name(),
			})

			resultsChan <- forecast
		}(repo)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Read all results from channel
	for forecast := range resultsChan {
		results[forecast.RepositoryName] = forecast
	}

	s.l.Info("completed forecast fetch", map[string]any{
		"results": results,
	})

	return results, nil
}
