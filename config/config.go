package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	App     AppConfig     `yaml:"app"`
	Server  ServerConfig  `yaml:"server"`
	Weather WeatherConfig `yaml:"weather"`
	Log     LogConfig     `yaml:"log"`
}

// AppConfig contains application-specific configuration
type AppConfig struct {
	Name    string `envconfig:"APP_NAME" yaml:"name" default:"weather-api"`
	Version string `envconfig:"APP_VERSION" yaml:"version" default:"1.0.0"`
	Env     string `envconfig:"APP_ENV" yaml:"env" default:"development"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         string `envconfig:"SERVER_PORT" yaml:"port" default:"8080"`
	ReadTimeout  int    `envconfig:"SERVER_READ_TIMEOUT" yaml:"read_timeout" default:"10"`
	WriteTimeout int    `envconfig:"SERVER_WRITE_TIMEOUT" yaml:"write_timeout" default:"10"`
	IdleTimeout  int    `envconfig:"SERVER_IDLE_TIMEOUT" yaml:"idle_timeout" default:"120"`
}

// WeatherConfig contains weather API configuration
type WeatherConfig struct {
	APIs []WeatherAPIConfig `yaml:"apis"`
}

// WeatherAPIConfig represents configuration for a weather API provider
type WeatherAPIConfig struct {
	Name    string `yaml:"name" validate:"required"`
	APIKey  string `yaml:"api_key,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`
	Timeout int    `yaml:"timeout" default:"30"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `envconfig:"LOG_LEVEL" yaml:"level" default:"info"`
	Format string `envconfig:"LOG_FORMAT" yaml:"format" default:"json"`
}

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	Load() (*Config, error)
	Validate(*Config) error
}

// FileConfigProvider loads configuration from files
type FileConfigProvider struct {
	configPath string
}

// NewFileConfigProvider creates a new file-based config provider
func NewFileConfigProvider(configPath string) *FileConfigProvider {
	return &FileConfigProvider{
		configPath: configPath,
	}
}

// Load loads configuration from YAML file and environment variables
func (p *FileConfigProvider) Load() (*Config, error) {
	config := &Config{}

	// Set default values
	if err := envconfig.Process("", config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	// Load from YAML file if it exists
	if err := p.loadFromFile(config); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Override with environment variables again (env vars take precedence)
	if err := envconfig.Process("", config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from YAML file
func (p *FileConfigProvider) loadFromFile(config *Config) error {
	// Try multiple possible config file locations
	configPaths := []string{
		p.configPath,
		"config/config.yaml",
		"config/config.yml",
		"./config/config.yaml",
		"./config/config.yml",
	}

	var configData []byte
	var err error

	for _, path := range configPaths {
		if configData, err = os.ReadFile(path); err == nil {
			break
		}
	}

	if err != nil {
		// Config file is optional, return without error
		return nil
	}

	if err := yaml.Unmarshal(configData, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (p *FileConfigProvider) Validate(config *Config) error {
	var errors []string

	// Validate App config
	if config.App.Name == "" {
		errors = append(errors, "app.name is required")
	}
	if config.App.Version == "" {
		errors = append(errors, "app.version is required")
	}

	// Validate Server config
	if config.Server.Port == "" {
		errors = append(errors, "server.port is required")
	}
	if config.Server.ReadTimeout <= 0 {
		errors = append(errors, "server.read_timeout must be positive")
	}
	if config.Server.WriteTimeout <= 0 {
		errors = append(errors, "server.write_timeout must be positive")
	}
	if config.Server.IdleTimeout <= 0 {
		errors = append(errors, "server.idle_timeout must be positive")
	}

	// Validate Weather APIs

	for i, api := range config.Weather.APIs {
		if api.Name == "" {
			errors = append(errors, fmt.Sprintf("weather.apis[%d].name is required", i))
		}
		if api.Timeout <= 0 {
			errors = append(errors, fmt.Sprintf("weather.apis[%d].timeout must be positive", i))
		}
	}

	// Validate Log config
	if config.Log.Level == "" {
		errors = append(errors, "log.level is required")
	}
	if config.Log.Format == "" {
		errors = append(errors, "log.format is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// NewConfig creates a new configuration instance
func NewConfig() (*Config, error) {
	return NewConfigWithProvider(NewFileConfigProvider("config/config.yaml"))
}

// NewConfigWithProvider creates a new configuration instance with a custom provider
func NewConfigWithProvider(provider ConfigProvider) (*Config, error) {
	config, err := provider.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := provider.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// GetWeatherAPIByName returns a weather API configuration by name
func (c *Config) GetWeatherAPIByName(name string) (*WeatherAPIConfig, bool) {
	for _, api := range c.Weather.APIs {
		if api.Name == name {
			return &api, true
		}
	}
	return nil, false
}

// GetWeatherAPIs returns all configured weather APIs
func (c *Config) GetWeatherAPIs() []WeatherAPIConfig {
	return c.Weather.APIs
}
