package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"weather-api/config"
	v1 "weather-api/internal/controllers/http/v1"
	"weather-api/internal/repositories"
	"weather-api/internal/services/weather"
	"weather-api/pkg/httpserver"
	"weather-api/pkg/observe"
)

// @title Weather API
// @version 1.0.0
// @description A high-performance, multi-provider weather forecast API built with Go and Fiber.
// @description This API aggregates weather data from multiple providers to provide reliable and comprehensive weather forecasts.
// @termsOfService http://swagger.io/terms/

// @contact.name Weather API Support
// @contact.url https://github.com/your-username/weather-api
// @contact.email support@weatherapi.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @tag.name Weather
// @tag.description Weather forecast operations
func main() {
	ctx, cancel := context.WithCancel(context.Background())

	cnf := config.NewConfig()

	l := observe.NewZapLogger(cnf.AppName, os.Stdout)

	app := httpserver.InitFiberServer(cnf.AppName)

	repos := repositories.InitWeatherRepositories(cnf, l)

	service := weather.NewWeatherService(repos, l)

	v1.NewRouter(
		app,
		service,
		l,
	)

	go func() {
		if err := app.Listen(":" + cnf.Port); err != nil {
			l.Fatal("cannot run the server", map[string]any{"err": err})
		}
	}()

	l.Info("application started successfully", map[string]any{"port": cnf.Port})

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		l.Warning("stopping application services")
		signal.Stop(sigCh)
		close(sigCh)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		_ = app.ShutdownWithContext(shutdownCtx)
		_ = l.Stop()
		cancel()
	}()

	select {
	case <-sigCh:
		fmt.Println("received shutdown signal")
	case <-ctx.Done():
		fmt.Println("context cancelled")
	}
}
