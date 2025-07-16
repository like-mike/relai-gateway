package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/helpers"
	"github.com/like-mike/relai-gateway/helpers/middleware"
	"github.com/like-mike/relai-gateway/routes/health"
	"github.com/like-mike/relai-gateway/routes/models"
	"github.com/like-mike/relai-gateway/routes/proxy"
)

func init() {
	os.Setenv("OTEL_SERVICE_NAME", "relai-gateway")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	os.Setenv("OTEL_TRACES_SAMPLER", "always_on")

}

func main() {
	// Load environment variables from .env file
	_ = godotenv.Load()

	tp := helpers.InitTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.CustomLogger())
	r.Use(gin.Recovery())

	// Public endpoints
	r.GET("/health", health.Handler)

	// Metrics endpoint (assuming /metrics is handled in prometheus middleware)
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.TracingMiddleware())

	// Protected group for all other routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddlewareGin())
	protected.GET("/v1/models", models.Handler)
	protected.GET("/models", models.Handler)

	// Fallback proxy route for all undefined HTTP requests (protected)
	r.NoRoute(func(c *gin.Context) {
		// Only allow health and metrics endpoints unauthenticated
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			proxy.Handler(c)
			return
		}
		// Require auth for all other undefined routes
		middleware.AuthMiddlewareGin()(c)
		if !c.IsAborted() {
			proxy.Handler(c)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Starting RelAI server on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
