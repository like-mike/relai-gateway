package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/middleware"
)

// SetupRouter initializes the Gin router and attaches handlers and middleware.
func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.CustomLogger())
	r.Use(gin.Recovery())

	// Prometheus metrics endpoint
	r.Use(middleware.PrometheusMiddleware())

	// Health endpoint
	r.GET("/health", HealthHandler)

	// OpenTelemetry middleware for tracing request body
	r.Use(middleware.TracingMiddleware())

	// Fallback proxy route for all undefined HTTP requests
	r.NoRoute(ProxyHandler)

	return r
}
