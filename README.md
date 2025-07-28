# Weather API

A high-performance, multi-provider weather forecast API built with Go and Fiber.

## Features

- 🌤️ **Multi-Provider Support**: OpenMeteo and/or WeatherAPI.com, you can easily extend the list of providers
- ⚡ **High Performance**: Built with Go and Fiber
- 🔄 **Concurrent Processing**: Fetches data from multiple providers simultaneously
- 🐳 **Docker Support**: Containerized deployment ready

## Quick Start

### Prerequisites
- Go 1.24.3 or higher
- Docker (optional)

### Local Development

1. **Clone and setup**
   ```bash
   git clone <repository-url>
   cd weather-api
   go mod download
   ```

2. **Configure providers**
   ```bash
   # Edit config/config.yaml to add API keys
   nano config/config.yaml
   ```

3. **Run the application**
   ```bash
   go run ./cmd/weather-api
   ```

### Docker

```bash
# Build and run
docker build -t weather-api .
docker run -p 8080:8080 weather-api
```

## API Usage

### Get Weather Forecast

**Endpoint:** `GET /weather`

**Parameters:**
- `lat` (required): Latitude (-90 to 90)
- `lon` (required): Longitude (-180 to 180)  
- `days` (optional): Forecast days (1-14, default: 5)

**Example:**
```bash
curl "http://localhost:8080/weather?lat=40.7128&lon=-74.0060&days=3"
```

**Response:**
```json
{
  "latitude": 40.7128,
  "longitude": -74.006,
  "forecast_window": 3,
  "forecasts": {
    "open-meteo": [
      {
        "date": "2025-07-28",
        "temp_max": 36.2,
        "temp_min": 23.5
      }
    ],
    "weatherapi": [
      {
        "date": "2025-07-28", 
        "temp_max": 35.1,
        "temp_min": 24.2
      }
    ]
  }
}
```

## Configuration

Edit `config/config.yaml`:

```yaml
app_name: "weather-api"
app_version: "1.0.0"
port: "8080"

weather_apis:
  - name: open-meteo
  - name: weatherapi
    api_key: "YOUR-API-KEY-HERE"
```

**Available Providers:**
- `open-meteo`: Free, no API key required
- `weatherapi`: Requires API key from [WeatherAPI.com](https://www.weatherapi.com/)

## Documentation

- **Swagger UI**: http://localhost:8080/swagger/
- **API Spec**: http://localhost:8080/swagger/doc.json

## Development

```bash
# Run tests
go test ./...

# Generate docs
./scripts/generate-docs.sh
```

## License

MIT License
