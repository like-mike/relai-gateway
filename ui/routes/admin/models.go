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

func ManageModelAccessHandler(c *gin.Context) {
	// This would handle updating model-organization access relationships
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message": "Model access management not yet implemented",
	})
}
