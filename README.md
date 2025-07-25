# Weather API

A high-performance, multi-provider weather forecast API built with Go and Fiber. This API aggregates weather data from multiple providers to provide reliable and comprehensive weather forecasts.

## Features

- 🌤️ **Multi-Provider Support**: Integrates with OpenMeteo and OpenWeatherMap APIs
- ⚡ **High Performance**: Built with Go and Fiber for optimal performance
- 🔄 **Concurrent Processing**: Fetches data from multiple providers simultaneously
- 🛡️ **Robust Error Handling**: Comprehensive error handling and validation
- 📊 **Structured Logging**: Detailed logging with Zap logger
- 🐳 **Docker Support**: Containerized deployment ready
- 📚 **API Documentation**: Swagger/OpenAPI documentation included

## Architecture

```
weather-api/
├── cmd/weather-api/          # Application entry point
├── config/                   # Configuration management
├── docs/                     # Generated Swagger documentation
├── internal/                 # Internal application code
│   ├── controllers/         # HTTP handlers and routing
│   ├── models/             # Data models
│   ├── repositories/       # Data access layer
│   └── services/           # Business logic
├── pkg/                     # Shared packages
│   ├── httpserver/         # HTTP server setup
│   ├── observe/            # Logging and observability
│   └── tools/              # Utility functions
├── scripts/                 # Utility scripts
└── Dockerfile              # Container configuration
```

## Quick Start

### Prerequisites

- Go 1.24.3 or higher
- Docker (optional)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd weather-api
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure the application**
   ```bash
   # Copy and edit the configuration
   cp config/config.yaml config/config.yaml.local
   ```

4. **Set environment variables**
   ```bash
   export PORT=8080
   export APP_NAME=weather-api
   # Add your OpenWeatherMap API key
   export WEATHERAPI_API_KEY=your-api-key-here
   ```

5. **Run the application**
   ```bash
   go run ./cmd/weather-api
   ```

### Docker Deployment

1. **Build the image**
   ```bash
   docker build -t weather-api .
   ```

2. **Run the container**
   ```bash
   docker run -p 8080:8080 \
     -e PORT=8080 \
     -e WEATHERAPI_API_KEY=your-api-key-here \
     weather-api
   ```

## API Documentation

### Swagger UI

The API documentation is available through Swagger UI:

- **Local Development**: http://localhost:8080/swagger/
- **Interactive Documentation**: Test API endpoints directly from the browser
- **Auto-generated**: Documentation is automatically generated from code annotations

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Get Weather Forecast

Retrieves weather forecast data for a specific location.

**Endpoint:** `GET /weather`

**Query Parameters:**
- `lat` (required): Latitude coordinate (-90 to 90)
- `lon` (required): Longitude coordinate (-180 to 180)
- `days` (optional): Number of forecast days (1-14, default: 5)

**Example Request:**
```bash
curl "http://localhost:8080/weather?lat=40.7128&lon=-74.0060&days=3"
```

**Example Response:**
```json
{
  "latitude": 40.7128,
  "longitude": -74.006,
  "forecast_window": 3,
  "forecasts": {
    "open-meteo": [
      {
        "date": "2025-07-25",
        "temp_max": 38.0,
        "temp_min": 24.3
      },
      {
        "date": "2025-07-26",
        "temp_max": 31.4,
        "temp_min": 23.6
      },
      {
        "date": "2025-07-27",
        "temp_max": 28.2,
        "temp_min": 22.4
      }
    ],
    "weatherapi": null
  }
}
```

**Error Responses:**

- `400 Bad Request`: Invalid parameters
  ```json
  {
    "error": "Missing required parameter: lat"
  }
  ```

- `500 Internal Server Error`: Service error
  ```json
  {
    "error": "Failed to fetch weather data"
  }
  ```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `APP_NAME` | Application name | `weather-api` |
| `APP_VERSION` | Application version | `1.0.0` |
| `WEATHERAPI_API_KEY` | OpenWeatherMap API key | - |

### Configuration File

The application uses `config/config.yaml` for provider configuration:

```yaml
app_name: "weather-api"
app_version: "1.0.0"
port: "8080"

weather_apis:
  - name: open-meteo
    base_url: "https://api.open-meteo.com/v1/forecast"
  - name: weatherapi
    base_url: "https://api.openweathermap.org/data/2.5/forecast"
    api_key: "YOUR-API-KEY-HERE"
```

## Weather Providers

### OpenMeteo
- **Provider**: Open-Meteo
- **API**: Free, no API key required
- **Features**: Global weather data, high accuracy
- **Rate Limits**: Generous limits

### OpenWeatherMap
- **Provider**: OpenWeatherMap
- **API**: Requires API key (free tier available)
- **Features**: Comprehensive weather data
- **Rate Limits**: Varies by plan

## Development

### Project Structure

```
internal/
├── controllers/           # HTTP request handling
│   └── http/v1/          # API version 1 handlers
├── models/               # Data structures
├── repositories/         # External API integrations
└── services/            # Business logic layer
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test package
go test ./internal/repositories/
```

### Generating Documentation

```bash
# Generate Swagger documentation
./scripts/generate-docs.sh

# Or manually
swag init -g cmd/weather-api/main.go -o docs
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Check for security vulnerabilities
gosec ./...
```

## Error Handling

The API implements comprehensive error handling:

- **Input Validation**: Validates all required parameters
- **HTTP Status Codes**: Proper HTTP status code responses
- **Provider Failures**: Graceful handling of provider failures
- **Detailed Logging**: Structured logging for debugging

### Common Error Scenarios

1. **Missing Parameters**: Returns 400 with specific error message
2. **Invalid Coordinates**: Returns 400 with validation error
3. **Provider Unavailable**: Continues with available providers
4. **API Rate Limits**: Handled gracefully with retry logic

## Monitoring and Logging

### Logging

The application uses structured logging with Zap:

```json
{
  "level": "info",
  "timestamp": "2025-07-25T19:34:07.800",
  "msg": "starting forecast fetch",
  "lat": 40.7128,
  "lon": -74.006,
  "forecastWindow": 5,
  "repositories": 2
}
```

### Metrics

Key metrics to monitor:
- Request latency
- Provider response times
- Error rates by provider
- API usage patterns

## Deployment

### Production Considerations

1. **Environment Variables**: Use proper environment variable management
2. **API Keys**: Secure storage of API keys
3. **Rate Limiting**: Implement rate limiting for API endpoints
4. **Health Checks**: Add health check endpoints
5. **Monitoring**: Set up monitoring and alerting

### Docker Compose

```yaml
version: '3.8'
services:
  weather-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - WEATHERAPI_API_KEY=${WEATHERAPI_API_KEY}
    restart: unless-stopped
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the API documentation
- Review the error logs for troubleshooting

## Roadmap

- [ ] Add more weather providers
- [ ] Implement caching layer
- [ ] Add historical weather data
- [ ] Implement rate limiting
- [ ] Add authentication
- [ ] Create client SDKs
