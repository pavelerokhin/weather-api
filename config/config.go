package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppName     string             `envconfig:"APP_NAME" default:"weather-api"`
	AppVersion  string             `envconfig:"APP_VERSION" default:"1.0.0"`
	Port        string             `envconfig:"PORT" default:"8080"`
	WeatherAPIs []WeatherAPIConfig `yaml:"weather_apis"`
}

type WeatherAPIConfig struct {
	Name   string `yaml:"name"`
	APIKey string `yaml:"api_key,omitempty"`
}

func NewConfig() *Config {
	var cnf Config

	// Read from YAML file first
	if yamlData, err := os.ReadFile("config/config.yaml"); err == nil {
		if err := yaml.Unmarshal(yamlData, &cnf); err != nil {
			panic(fmt.Sprintf("Warning: failed to parse YAML config: %v\n", err))
		}
	}

	// Override with environment variables
	if err := envconfig.Process("", &cnf); err != nil {
		panic(fmt.Errorf("error environment variable parsing: %w", err))
	}

	return &cnf
}
