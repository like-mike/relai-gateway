package admin

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

func ModelsHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get models from database with organization access
	modelsList, err := db.GetModelsWithOrganizations(sqlDB)
	if err != nil {
		log.Printf("Failed to get models: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load models"})
		return
	}

	// Return JSON response for JavaScript to render
	c.JSON(http.StatusOK, models.ModelsResponse{
		Models: modelsList,
	})
}

func CreateModelHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Parse JSON request
	var req models.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind model request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Create model in database
	model, err := db.CreateModel(sqlDB, req)
	if err != nil {
		log.Printf("Failed to create model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create model"})
		return
	}

	// Return the created model
	c.JSON(http.StatusCreated, gin.H{
		"model":   model,
		"message": "Model created successfully",
	})
}

func DeleteModelHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get model ID from URL parameter
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	// Delete model (soft delete)
	err := db.DeleteModel(sqlDB, modelID)
	if err != nil {
		log.Printf("Failed to delete model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Model deleted successfully",
	})
}

func UpdateModelHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get model ID from URL parameter
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	// Parse JSON request
	var req models.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind model update request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update model in database
	model, err := db.UpdateModel(sqlDB, modelID, req)
	if err != nil {
		log.Printf("Failed to update model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
		return
	}

	// Return the updated model
	c.JSON(http.StatusOK, gin.H{
		"model":   model,
		"message": "Model updated successfully",
	})
}

func ManageModelAccessHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get model ID from URL parameter
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
		return
	}

	// Parse JSON request
	var req struct {
		Changes []db.ModelAccessChange `json:"changes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind access management request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update model access in database
	err := db.ManageModelAccess(sqlDB, modelID, req.Changes)
	if err != nil {
		log.Printf("Failed to manage model access: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model access"})
		return
	}

	// Get the updated model with organizations for the response
	model, err := db.GetModelWithOrganizations(sqlDB, modelID)
	if err != nil {
		log.Printf("Failed to get updated model: %v", err)
		// Still return success since the access was updated
		c.JSON(http.StatusOK, gin.H{
			"message": "Model access updated successfully",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"model":   model,
		"message": "Model access updated successfully",
	})
}

// Endpoints handlers
func EndpointsHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get endpoints from database
	endpointsList, err := db.GetEndpointsWithModels(sqlDB)
	if err != nil {
		log.Printf("Failed to get endpoints: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load endpoints"})
		return
	}

	// Return JSON response
	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpointsList,
	})
}

func CreateEndpointHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Parse JSON request
	var req models.EndpointCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind endpoint request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get organization ID from request or use a default (in real app, this would come from user context)
	orgID := c.PostForm("organization_id")
	if orgID == "" {
		// For demo, use the first organization
		orgID = "11111111-1111-1111-1111-111111111111"
	}

	// Create endpoint in database
	endpoint, err := db.CreateEndpoint(sqlDB, req, orgID)
	if err != nil {
		log.Printf("Failed to create endpoint: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create endpoint"})
		return
	}

	// Return the created endpoint
	c.JSON(http.StatusCreated, gin.H{
		"endpoint": endpoint,
		"message":  "Endpoint created successfully",
	})
}

func UpdateEndpointHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get endpoint ID from URL parameter
	endpointID := c.Param("id")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Endpoint ID is required"})
		return
	}

	// Parse JSON request
	var req models.EndpointUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind endpoint update request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update endpoint in database
	endpoint, err := db.UpdateEndpoint(sqlDB, endpointID, req)
	if err != nil {
		log.Printf("Failed to update endpoint: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update endpoint"})
		return
	}

	// Return the updated endpoint
	c.JSON(http.StatusOK, gin.H{
		"endpoint": endpoint,
		"message":  "Endpoint updated successfully",
	})
}

func DeleteEndpointHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get endpoint ID from URL parameter
	endpointID := c.Param("id")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Endpoint ID is required"})
		return
	}

	// Delete endpoint (soft delete)
	err := db.DeleteEndpoint(sqlDB, endpointID)
	if err != nil {
		log.Printf("Failed to delete endpoint: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete endpoint"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Endpoint deleted successfully",
	})
}

func GetEndpointHandler(c *gin.Context) {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	// Get endpoint ID from URL parameter
	endpointID := c.Param("id")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Endpoint ID is required"})
		return
	}

	// Get endpoint from database
	endpoint, err := db.GetEndpointByID(sqlDB, endpointID)
	if err != nil {
		log.Printf("Failed to get endpoint: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get endpoint"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint": endpoint,
	})
}
