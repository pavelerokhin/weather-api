package models

import "fmt"

type Forecast struct {
	RepositoryName string        `json:"repository_name" example:"openmeteo"`
	Lat            float64       `json:"lat" example:"40.7128"`
	Lon            float64       `json:"lon" example:"-74.006"`
	ForecastWindow int           `json:"forecast_window" example:"5"`
	ForecastData   []WeatherData `json:"forecast_data"`
}

func (f *Forecast) RequestParams() string {
	return fmt.Sprintf("lat: %.4f lon: %.4f days: %d", f.Lat, f.Lon, f.ForecastWindow)
}
