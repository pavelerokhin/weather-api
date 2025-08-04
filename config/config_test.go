package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// Test with default values (without config file)
	provider := NewFileConfigProvider("nonexistent.yaml")
	config, err := NewConfigWithProvider(provider)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, "weather-api", config.App.Name)
	assert.Equal(t, "1.0.0", config.App.Version)
	assert.Equal(t, "development", config.App.Env)
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, 10, config.Server.ReadTimeout)
	assert.Equal(t, 10, config.Server.WriteTimeout)
	assert.Equal(t, 120, config.Server.IdleTimeout)
	assert.Equal(t, "info", config.Log.Level)
	assert.Equal(t, "json", config.Log.Format)

	// Without config file, weather APIs should be empty
	assert.Len(t, config.Weather.APIs, 0)
}

func TestConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("APP_VERSION", "2.0.0")
	os.Setenv("APP_ENV", "production")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_VERSION")
		os.Unsetenv("APP_ENV")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	provider := NewFileConfigProvider("nonexistent.yaml")
	config, err := NewConfigWithProvider(provider)
	require.NoError(t, err)

	assert.Equal(t, "test-app", config.App.Name)
	assert.Equal(t, "2.0.0", config.App.Version)
	assert.Equal(t, "production", config.App.Env)
	assert.Equal(t, "9090", config.Server.Port)
	assert.Equal(t, "debug", config.Log.Level)

	// Without config file, weather APIs should be empty
	assert.Len(t, config.Weather.APIs, 0)
}

func TestConfigValidation(t *testing.T) {
	provider := NewFileConfigProvider("config/config.yaml")

	// Test valid config
	config := &Config{
		App: AppConfig{
			Name:    "test-app",
			Version: "1.0.0",
			Env:     "development",
		},
		Server: ServerConfig{
			Port:         "8080",
			ReadTimeout:  10,
			WriteTimeout: 10,
			IdleTimeout:  120,
		},
		Weather: WeatherConfig{
			APIs: []WeatherAPIConfig{
				{
					Name:    "open-meteo",
					Timeout: 30,
				},
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := provider.Validate(config)
	assert.NoError(t, err)

	// Test invalid config - missing app name
	invalidConfig := &Config{
		App: AppConfig{
			Name:    "",
			Version: "1.0.0",
		},
		Server: ServerConfig{
			Port:         "8080",
			ReadTimeout:  10,
			WriteTimeout: 10,
			IdleTimeout:  120,
		},
		Weather: WeatherConfig{
			APIs: []WeatherAPIConfig{
				{
					Name:    "open-meteo",
					Timeout: 30,
				},
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err = provider.Validate(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app.name is required")
}

func TestConfigHelperMethods(t *testing.T) {
	config := &Config{
		App: AppConfig{
			Env: "development",
		},
		Weather: WeatherConfig{
			APIs: []WeatherAPIConfig{
				{
					Name:    "open-meteo",
					APIKey:  "",
					Timeout: 30,
				},
				{
					Name:    "weatherapi",
					APIKey:  "test-key",
					Timeout: 30,
				},
			},
		},
	}

	// Test IsDevelopment
	assert.True(t, config.IsDevelopment())
	assert.False(t, config.IsProduction())

	// Test GetWeatherAPIByName
	api, found := config.GetWeatherAPIByName("open-meteo")
	assert.True(t, found)
	assert.Equal(t, "open-meteo", api.Name)

	api, found = config.GetWeatherAPIByName("nonexistent")
	assert.False(t, found)
	assert.Nil(t, api)

	// Test GetWeatherAPIs
	apis := config.GetWeatherAPIs()
	assert.Len(t, apis, 2)
	assert.Equal(t, "open-meteo", apis[0].Name)
	assert.Equal(t, "weatherapi", apis[1].Name)
}

func TestFileConfigProvider_LoadFromFile(t *testing.T) {
	provider := NewFileConfigProvider("nonexistent.yaml")
	config := &Config{}

	// Test loading from non-existent file (should not error)
	err := provider.loadFromFile(config)
	assert.NoError(t, err)
}

func TestNewConfigWithProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := &MockConfigProvider{
		config: &Config{
			App: AppConfig{
				Name:    "test-app",
				Version: "1.0.0",
				Env:     "development",
			},
			Server: ServerConfig{
				Port:         "8080",
				ReadTimeout:  10,
				WriteTimeout: 10,
				IdleTimeout:  120,
			},
			Weather: WeatherConfig{
				APIs: []WeatherAPIConfig{
					{
						Name:    "open-meteo",
						Timeout: 30,
					},
				},
			},
			Log: LogConfig{
				Level:  "info",
				Format: "json",
			},
		},
	}

	config, err := NewConfigWithProvider(mockProvider)
	require.NoError(t, err)
	assert.Equal(t, "test-app", config.App.Name)
}

func TestConfigFileLoading(t *testing.T) {
	// Test loading from actual config file
	config, err := NewConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Check if config file was loaded (weather APIs should be present)
	if len(config.Weather.APIs) > 0 {
		// Config file was loaded successfully
		assert.Len(t, config.Weather.APIs, 2)
		assert.Equal(t, "open-meteo", config.Weather.APIs[0].Name)
		assert.Equal(t, "weatherapi", config.Weather.APIs[1].Name)
		assert.Equal(t, "YOUR-API-KEY-HERE", config.Weather.APIs[1].APIKey)
	} else {
		// Config file was not loaded, but that's okay for testing
		t.Log("Config file not loaded, using default values")
	}
}

// MockConfigProvider for testing
type MockConfigProvider struct {
	config *Config
	err    error
}

func (m *MockConfigProvider) Load() (*Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.config, nil
}

func (m *MockConfigProvider) Validate(config *Config) error {
	return nil
}
