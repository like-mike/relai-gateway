package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/provider"
	"github.com/like-mike/relai-gateway/routes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Remove duplicate CompletionsRequest, CompletionsResponse, and main function
// load environment variables from .env
func main() {
	_ = godotenv.Load()

	provider, err := provider.NewProviderFromEnv()
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	app := fiber.New()
	// Add Prometheus metrics middleware
	app.Use(routes.PrometheusMiddleware())
	// Add HTTP logging middleware
	app.Use(logger.New())
	// Expose Prometheus metrics at /metrics
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	routes.RegisterRoutes(app, provider)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Starting server on :" + port)
	log.Fatal(app.Listen(":" + port))
}
