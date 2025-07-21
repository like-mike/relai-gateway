package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/like-mike/relai-gateway/gateway/routes/health"
	"github.com/like-mike/relai-gateway/gateway/routes/models"
	"github.com/like-mike/relai-gateway/gateway/routes/proxy"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/middleware"
	"github.com/like-mike/relai-gateway/shared/tracer"
)

func main() {
	// Load environment variables
	_ = godotenv.Load("../.env")

	// Initialize DB
	conn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Seed data for testing
	if err := db.SeedTestData(conn); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	// Initialize OpenTelemetry tracer
	tp := tracer.InitTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Setup Gin router
	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.CustomLogger())
	r.Use(gin.Recovery())

	// Attach DB to Gin context
	r.Use(middleware.DBMiddleware(conn))
	// Static health check
	r.GET("/health", health.Handler)

	// Prometheus and tracing
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.TracingMiddleware())

	// Public model routes
	r.GET("/v1/models", models.Handler)
	r.GET("/models", models.Handler)

	// Protected routes

	// Catch-all for proxying
	r.NoRoute(proxy.Handler)

	// Run server
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting RelAI server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
