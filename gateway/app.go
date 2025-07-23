package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/like-mike/relai-gateway/gateway/middleware"
	"github.com/like-mike/relai-gateway/gateway/routes/health"
	"github.com/like-mike/relai-gateway/gateway/routes/models"
	"github.com/like-mike/relai-gateway/gateway/routes/proxy"
	"github.com/like-mike/relai-gateway/shared/db"
	sharedmw "github.com/like-mike/relai-gateway/shared/middleware"
	"github.com/like-mike/relai-gateway/shared/tracer"
	"github.com/like-mike/relai-gateway/shared/usage"
)

// getUsageConfig returns usage tracking configuration from environment variables
func getUsageConfig() *usage.WorkerConfig {
	config := usage.DefaultWorkerConfig()

	// Override with environment variables if set
	if workerCountStr := os.Getenv("USAGE_WORKER_COUNT"); workerCountStr != "" {
		if count, err := strconv.Atoi(workerCountStr); err == nil && count > 0 {
			config.WorkerCount = count
		}
	}

	if queueSizeStr := os.Getenv("USAGE_QUEUE_SIZE"); queueSizeStr != "" {
		if size, err := strconv.Atoi(queueSizeStr); err == nil && size > 0 {
			config.QueueSize = size
		}
	}

	if maxRetriesStr := os.Getenv("USAGE_MAX_RETRIES"); maxRetriesStr != "" {
		if retries, err := strconv.Atoi(maxRetriesStr); err == nil && retries >= 0 {
			config.MaxRetries = retries
		}
	}

	if retryDelayStr := os.Getenv("USAGE_RETRY_DELAY"); retryDelayStr != "" {
		if delay, err := time.ParseDuration(retryDelayStr); err == nil {
			config.RetryDelay = delay
		}
	}

	// Check if usage tracking is disabled
	if disabled := os.Getenv("USAGE_TRACKING_DISABLED"); disabled == "true" || disabled == "1" {
		log.Println("Usage tracking disabled via environment variable")
		config.WorkerCount = 0 // Disable workers
	}

	return config
}

func main() {
	// Load environment variables
	_ = godotenv.Load("../.env")

	// Initialize DB
	conn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Initialize OpenTelemetry tracer
	tp := tracer.InitTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Initialize usage tracking
	usageConfig := getUsageConfig()
	usage.InitGlobalUsageTracker(conn, usageConfig)
	defer usage.StopGlobalUsageTracker()
	log.Printf("Usage tracking initialized with %d workers", usageConfig.WorkerCount)

	// Setup Gin router
	r := gin.New()
	r.Use(sharedmw.CORSMiddleware())
	r.Use(sharedmw.CustomLogger())
	r.Use(gin.Recovery())

	// Attach DB to Gin context
	r.Use(sharedmw.DBMiddleware(conn))

	// Static health check (no auth required)
	r.GET("/health", health.Handler)

	// Prometheus and tracing
	r.Use(sharedmw.PrometheusMiddleware())
	r.Use(sharedmw.TracingMiddleware())

	// Public model routes (optional auth - works with or without API key)
	r.GET("/v1/models", middleware.OptionalAPIKeyAuth(), models.Handler)
	r.GET("/models", middleware.OptionalAPIKeyAuth(), models.Handler)

	// Standard OpenAI API pass-through routes (requires API key from database)
	api := r.Group("/v1")
	api.Use(middleware.APIKeyAuth()) // Requires valid API key from database
	{
		// Standard OpenAI API endpoints
		api.POST("/chat/completions", proxy.Handler)
		api.POST("/completions", proxy.Handler)
		api.POST("/embeddings", proxy.Handler)
		api.POST("/moderations", proxy.Handler)
		api.POST("/images/generations", proxy.Handler)
		api.POST("/audio/transcriptions", proxy.Handler)
		api.POST("/audio/translations", proxy.Handler)
	}

	// Protected routes group (requires API key authentication)
	protected := r.Group("/")
	protected.Use(middleware.APIKeyAuth())
	{
		// Removed completions proxy endpoint registration
		// Add any other protected endpoints here in the future
	}

	// Custom endpoints and catch-all - requires API key from database
	// This handles both custom organization endpoints and any other API calls
	r.NoRoute(middleware.APIKeyAuth(), proxy.Handler)

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
