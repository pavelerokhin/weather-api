package httpserver

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func InitFiberServer(appName string) *fiber.App {
	s := fiber.New(fiber.Config{
		AppName:           appName,
		JSONEncoder:       json.Marshal,
		JSONDecoder:       json.Unmarshal,
		BodyLimit:         500 * 1024 * 1024,
		StreamRequestBody: true,
	})

	s.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	s.Use(cors.New())
	s.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/manage/health",
		ReadinessEndpoint: "/manage/ready",
	}))

	return s
}
