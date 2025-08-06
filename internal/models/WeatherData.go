package models

import "time"

type WeatherData struct {
	Date    *time.Time `json:"date" example:"2023-10-01"`
	TempMax float64    `json:"temp_max" example:"38.0"`
	TempMin float64    `json:"temp_min" example:"24.3"`
}

// FilterByDate returns the index of the WeatherData with the matching date, or -1 if not found
func FilterByDate(data []WeatherData, date *time.Time) int {
	for i, wd := range data {
		if wd.Date != nil && date != nil && wd.Date.Equal(*date) {
			return i
		}
	}
	return -1
}
