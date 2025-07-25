#!/bin/bash

# Generate Swagger documentation for the Weather API
echo "Generating Swagger documentation..."

# Check if swag is installed
if ! command -v swag &> /dev/null; then
    echo "swag is not installed. Installing..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Generate docs
swag init -g cmd/weather-api/main.go -o docs

echo "Swagger documentation generated successfully!"
echo "You can now access the documentation at: http://localhost:8080/swagger/" 