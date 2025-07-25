package models

type Response struct {
	Date    string  `json:"date"`
	TempMax float64 `json:"temp_max"`
	TempMin float64 `json:"temp_min"`
}
