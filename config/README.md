# Configuration

This package implements a clean architecture configuration system following the principles from [go-clean-template](https://github.com/evrone/go-clean-template).

## Features

- **Interface-based design** - Configurable through `ConfigProvider` interface
- **Multiple sources** - Support for YAML files and environment variables
- **Validation** - Comprehensive configuration validation
- **Environment-specific** - Support for different environments (development, production)
- **Testable** - Easy to mock and test with custom providers
- **Error handling** - Proper error handling instead of panics

## Structure

### Config Sections

The configuration is organized into logical sections:

```go
type Config struct {
    App      AppConfig      // Application metadata
    Server   ServerConfig   // HTTP server settings
    Weather  WeatherConfig  // Weather API providers
    Log      LogConfig      // Logging configuration
}
```

### Configuration Hierarchy

1. **Default values** - Set in struct tags
2. **YAML file** - Loaded from `config/config.yaml`
3. **Environment variables** - Override YAML values (highest priority)

## Usage

### Basic Usage

```go
import "weather-api/config"

// Load configuration with default provider
cfg, err := config.NewConfig()
if err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Println("App name:", cfg.App.Name)
fmt.Println("Server port:", cfg.Server.Port)
```

### Custom Provider

```go
// Create custom provider
provider := config.NewFileConfigProvider("custom-config.yaml")
cfg, err := config.NewConfigWithProvider(provider)
if err != nil {
    log.Fatal(err)
}
```

### Environment Variables

All configuration can be overridden with environment variables:

```bash
export APP_NAME="my-weather-api"
export APP_ENV="production"
export SERVER_PORT="9090"
export LOG_LEVEL="debug"
```

## Configuration File

### YAML Structure

```yaml
app:
  name: "weather-api"
  version: "1.0.0"
  env: "development"

server:
  port: "8080"
  read_timeout: 10
  write_timeout: 10
  idle_timeout: 120

weather:
  apis:
    - name: open-meteo
      timeout: 30
    - name: weatherapi
      api_key: "YOUR-API-KEY-HERE"
      timeout: 30

log:
  level: "info"
  format: "json"
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_NAME` | Application name | `weather-api` |
| `APP_VERSION` | Application version | `1.0.0` |
| `APP_ENV` | Environment | `development` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `SERVER_READ_TIMEOUT` | Read timeout (seconds) | `10` |
| `SERVER_WRITE_TIMEOUT` | Write timeout (seconds) | `10` |
| `SERVER_IDLE_TIMEOUT` | Idle timeout (seconds) | `120` |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_FORMAT` | Log format | `json` |

## Testing

### Unit Tests

```go
// Test with mock provider
mockProvider := &MockConfigProvider{
    config: &Config{
        App: AppConfig{Name: "test-app"},
        // ... other config
    },
}

cfg, err := NewConfigWithProvider(mockProvider)
require.NoError(t, err)
assert.Equal(t, "test-app", cfg.App.Name)
```

### Integration Tests

```go
// Test with actual config file
cfg, err := NewConfig()
require.NoError(t, err)
assert.NotNil(t, cfg)
```

## Validation

The configuration system includes comprehensive validation:

- Required fields (app name, version, server port)
- Positive timeouts and intervals
- Weather API configuration
- Log level and format validation

## Clean Architecture Principles

This configuration system follows clean architecture principles:

1. **Dependency Inversion** - Uses `ConfigProvider` interface
2. **Single Responsibility** - Each config section has a specific purpose
3. **Open/Closed Principle** - Easy to extend with new providers
4. **Interface Segregation** - Minimal interface requirements
5. **Dependency Injection** - Configurable through providers

## Migration from Old Config

If you're migrating from the old config structure:

### Old Structure
```go
type Config struct {
    AppName     string
    AppVersion  string
    Port        string
    WeatherAPIs []WeatherAPIConfig
}
```

### New Structure
```go
type Config struct {
    App      AppConfig
    Server   ServerConfig
    Weather  WeatherConfig
    Log      LogConfig
}
```

### Access Changes
```go
// Old
cfg.AppName
cfg.Port
cfg.WeatherAPIs

// New
cfg.App.Name
cfg.Server.Port
cfg.Weather.APIs
```

## Best Practices

1. **Use environment variables for secrets** - Never commit API keys to YAML
2. **Validate configuration early** - Check config at startup
3. **Use typed configuration** - Leverage Go's type system
4. **Test configuration loading** - Ensure config works in all environments
5. **Document configuration options** - Keep this README updated 