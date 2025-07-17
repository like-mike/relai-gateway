package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/middleware"
	"github.com/like-mike/relai-gateway/ui/routes/admin"
	"github.com/like-mike/relai-gateway/ui/routes/health"
)

func main() {
	// Load environment variables
	_ = godotenv.Load("../.env")
	config := admin.LoadAuthConfig()

	// Initialize DB
	conn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer conn.Close()

	// Setup Gin router
	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.CustomLogger())
	r.Use(gin.Recovery())

	// Serve templates including partials
	r.LoadHTMLGlob("templates/*.html")

	// Attach DB to Gin context
	r.Use(middleware.DBMiddleware(conn))
	// Static health check
	r.GET("/health", health.Handler)
	// Register public authentication routes
	admin.RegisterPublicAuthRoutes(r, config)

	// Protected routes
	authorized := r.Group("/")
	authorized.Use(middleware.AuthMiddlewareGin())
	admin.RegisterAuthRoutes(authorized, config)

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

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	// Run server
	port := os.Getenv("UI_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting RelAI UI server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
