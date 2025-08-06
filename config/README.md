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
