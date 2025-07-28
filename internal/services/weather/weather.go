package weather

import (
	"context"
	"sync"

	"weather-api/internal/models"
	"weather-api/internal/repositories"
	"weather-api/pkg/observe"
)

// WeatherService represents the weather service.
type WeatherService struct {
	repos []repositories.WeatherRepository
	l     *observe.Logger
}

func NewWeatherService(repos []repositories.WeatherRepository, l *observe.Logger) *WeatherService {
	return &WeatherService{
		repos: repos,
		l:     l,
	}
}

// FetchForecasts fetches the weather forecasts from all available APIs for the given latitude and longitude
func (s *WeatherService) FetchForecasts(ctx context.Context, lat, lon float64, forecastWindow int) (map[string][]models.Response, error) {
	s.l.Info("starting forecast fetch", map[string]any{
		"lat":            lat,
		"lon":            lon,
		"forecastWindow": forecastWindow,
		"repositories":   len(s.repos),
	})

	results := make(map[string][]models.Response)
	var mu sync.Mutex

	wg := sync.WaitGroup{}

	for _, repo := range s.repos {
		wg.Add(1)

		go func(repo repositories.WeatherRepository) {
			defer wg.Done()
			s.l.Debug("fetching forecast", map[string]any{"repo": repo.Name(), "lat": lat, "lon": lon})

			forecast, err := repo.FetchForecast(ctx, lat, lon, forecastWindow)
			if err != nil {
				mu.Lock()
				results[repo.Name()] = []models.Response{}
				mu.Unlock()

				s.l.Error(err, map[string]any{"repo": repo.Name(), "err": err})

				return
			}

			mu.Lock()
			results[repo.Name()] = forecast
			mu.Unlock()

			s.l.Info("successfully fetched forecast", map[string]any{
				"repo": repo.Name(),
				"days": len(forecast),
			})
		}(repo)
	}

	wg.Wait()

	s.l.Info("completed forecast fetch", map[string]any{
		"results": results,
	})

	if len(results) == 0 {
		s.l.Warning("no results found")
		return nil, nil
	}

	return results, nil
}
