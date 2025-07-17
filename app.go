package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/internal/db"
	"github.com/like-mike/relai-gateway/internal/helpers"
	"github.com/like-mike/relai-gateway/internal/helpers/middleware"
	"github.com/like-mike/relai-gateway/internal/routes/admin"
	"github.com/like-mike/relai-gateway/internal/routes/health"
	"github.com/like-mike/relai-gateway/internal/routes/models"
	"github.com/like-mike/relai-gateway/internal/routes/proxy"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()
	config := admin.LoadAuthConfig()

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
	tp := helpers.InitTracer()
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

	// Serve templates including partials
	r.LoadHTMLGlob("web/templates/*.html")

	// Attach DB to Gin context
	r.Use(middleware.DBMiddleware(conn))

	// Admin API routes (moved to authorized group)

	// Static health check
	r.GET("/health", health.Handler)

	// Prometheus and tracing
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.TracingMiddleware())

	// Public model routes
	r.GET("/v1/models", models.Handler)
	r.GET("/models", models.Handler)

	// // Register public authentication routes from admin package
	// admin.RegisterPublicAuthRoutes(r, config)

	// Register public authentication routes
	admin.RegisterPublicAuthRoutes(r, config)

	// Protected routes
	authorized := r.Group("/")
	authorized.Use(middleware.AuthMiddlewareGin())
	admin.RegisterAuthRoutes(authorized, config)
	// authorized.GET("/admin/logout", func(c *gin.Context) {
	// 	fmt.Println("Logging out user")
	// 	c.SetCookie("session", "", -1, "/", "", false, true)
	// 	c.Redirect(http.StatusFound, "/admin")
	// })
	authorized.GET("/admin", admin.DashboardHandler)
	authorized.GET("/admin/test-api", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test_api.html", gin.H{"activePage": "test_api", "isAuthenticated": true})
	})
	authorized.GET("/quota", func(c *gin.Context) {
		admin.GetQuotaHandler(c)
		c.Set("isAuthenticated", true)
	})
	authorized.GET("/api-keys", func(c *gin.Context) {
		admin.APIKeysHandler(c)
		// admin.GetDropdownHandler(c)
		c.Set("isAuthenticated", true)
	})

	// Admin panel UI
	// (removed duplicate, now only in authorized group)

	// Test API page route
	// (removed duplicate, now only in authorized group)

	// Redirect root to /admin
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	// Catch-all for proxying
	r.NoRoute(proxy.Handler)

	// Run server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting RelAI server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
