package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/shared/config"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/middleware"
	uimw "github.com/like-mike/relai-gateway/ui/middleware"
	"github.com/like-mike/relai-gateway/ui/routes/admin"
	"github.com/like-mike/relai-gateway/ui/routes/health"
)

func main() {
	// Load environment variables
	_ = godotenv.Load("../.env")
	authConfig := admin.LoadAuthConfig()

	// Load theme configuration
	_, err := config.LoadConfig("../config.yml")
	if err != nil {
		log.Printf("Warning: Failed to load theme config: %v", err)
	}

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

	// Load templates using LoadHTMLFiles to avoid conflicts
	templateFiles := []string{
		"templates/pages/auth/login.html",
		"templates/pages/admin/api-keys.html",
		"templates/pages/admin/models.html",
		"templates/pages/admin/analytics.html",
		"templates/pages/admin/test-api.html",
		"templates/pages/admin/settings.html",
		"templates/pages/admin/docs.html",
		"templates/components/ui/banner.html",
		"templates/components/ui/sidebar.html",
		"templates/components/ui/user-dropdown.html",
		"templates/partials/org-selector.html",
		"templates/partials/quota-cards.html",
		"templates/partials/api-keys-table.html",
		"templates/partials/organizations-table.html",
		"templates/partials/users-table.html",
		"templates/components/modals/api-keys/new-key-modal.html",
		"templates/components/modals/api-keys/view-key-modal.html",
		"templates/components/modals/api-keys/delete-confirmation-modal.html",
		"templates/components/modals/models/add-model-modal.html",
		"templates/components/modals/models/edit-model-modal.html",
		"templates/components/modals/models/delete-model-modal.html",
		"templates/components/modals/models/manage-access-modal.html",
		"templates/components/modals/endpoints/add-endpoint-modal.html",
		"templates/components/modals/endpoints/edit-endpoint-modal.html",
		"templates/components/modals/endpoints/delete-endpoint-modal.html",
		"templates/components/modals/organizations/create-org-modal.html",
		"templates/components/modals/organizations/edit-org-modal.html",
		"templates/shared/theme.css",
	}
	r.LoadHTMLFiles(templateFiles...)

	// Attach DB to Gin context
	r.Use(middleware.DBMiddleware(conn))

	// Health check
	r.GET("/health", health.Handler)

	// Dynamic theme CSS endpoint
	r.GET("/theme.css", func(c *gin.Context) {
		userData := admin.GetUserContext(c)
		c.Header("Content-Type", "text/css")
		c.HTML(http.StatusOK, "theme.css", userData)
	})

	// Serve docs directory files publicly (for Swagger UI to fetch)
	r.Static("/docs", "../docs")

	// Register public authentication routes
	admin.RegisterPublicAuthRoutes(r, authConfig)

	// Root route redirect
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	// Protected routes
	authorized := r.Group("/")
	authorized.Use(uimw.AuthMiddlewareGin())
	admin.RegisterAuthRoutes(authorized, authConfig)

	// Admin dashboard - API Keys page
	authorized.GET("/admin", admin.DashboardHandler)
	authorized.GET("/admin/models", func(c *gin.Context) {
		userData := admin.GetUserContext(c)
		userData["activePage"] = "models"
		userData["title"] = "Models Management"
		c.HTML(http.StatusOK, "models.html", userData)
	})
	authorized.GET("/admin/test-api", func(c *gin.Context) {
		userData := admin.GetUserContext(c)
		userData["activePage"] = "test_api"
		userData["title"] = "Test API"
		c.HTML(http.StatusOK, "test-api.html", userData)
	})
	authorized.GET("/admin/settings", admin.SettingsHandler)
	authorized.GET("/admin/analytics", func(c *gin.Context) {
		userData := admin.GetUserContext(c)
		userData["activePage"] = "analytics"
		userData["title"] = "Usage Analytics"
		c.HTML(http.StatusOK, "analytics.html", userData)
	})
	authorized.GET("/admin/docs", func(c *gin.Context) {
		userData := admin.GetUserContext(c)
		userData["activePage"] = "docs"
		userData["title"] = "API Documentation"
		c.HTML(http.StatusOK, "docs.html", userData)
	})

	// API endpoints with database integration
	authorized.GET("/quota", admin.GetQuotaHandler)
	authorized.GET("/api-keys", admin.APIKeysHandler)
	authorized.POST("/api/keys", admin.CreateAPIKeyHandler)
	authorized.DELETE("/api/keys/:id", admin.DeleteAPIKeyHandler)
	authorized.GET("/api/organizations", admin.OrganizationsHandler)
	authorized.GET("/api/models", admin.ModelsHandler)
	authorized.POST("/api/models", admin.CreateModelHandler)
	authorized.PUT("/api/models/:id", admin.UpdateModelHandler)
	authorized.DELETE("/api/models/:id", admin.DeleteModelHandler)
	authorized.POST("/api/models/:id/access", admin.ManageModelAccessHandler)
	authorized.GET("/api/analytics/dashboard", admin.AnalyticsDashboardHandler)
	authorized.POST("/api/completions-proxy", admin.CompletionsProxyHandler)

	// Endpoints API routes
	authorized.GET("/api/endpoints", admin.EndpointsHandler)
	authorized.POST("/api/endpoints", admin.CreateEndpointHandler)
	authorized.GET("/api/endpoints/:id", admin.GetEndpointHandler)
	authorized.PUT("/api/endpoints/:id", admin.UpdateEndpointHandler)
	authorized.DELETE("/api/endpoints/:id", admin.DeleteEndpointHandler)

	// Settings endpoints
	authorized.GET("/admin/settings/organizations", admin.OrganizationsTableHandler)
	authorized.POST("/admin/settings/organizations", admin.CreateOrganizationHandler)
	authorized.GET("/admin/settings/organizations/:id", admin.GetOrganizationHandler)
	authorized.PUT("/admin/settings/organizations/:id", admin.UpdateOrganizationHandler)
	authorized.POST("/admin/settings/organizations/:id", admin.UpdateOrganizationHandler) // HTMX form support
	authorized.DELETE("/admin/settings/organizations/:id", admin.DeleteOrganizationHandler)
	authorized.GET("/admin/settings/users", admin.UsersTableHandler)
	authorized.GET("/admin/settings/ad-groups", admin.GetADGroupsHandler)

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
