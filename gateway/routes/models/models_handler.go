package models

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/db"
)

// ModelsResponse represents the OpenAI-compatible models response
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents an OpenAI-compatible model
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func Handler(c *gin.Context) {
	// Check if user is authenticated and has accessible models
	accessibleModelsInterface, hasAuth := c.Get("accessible_models")
	if hasAuth {
		// User is authenticated, return only models they have access to
		if accessibleModels, ok := accessibleModelsInterface.([]AccessibleModel); ok {
			var models []Model
			for _, accessibleModel := range accessibleModels {
				if accessibleModel.IsActive {
					models = append(models, Model{
						ID:      accessibleModel.ModelID, // Use the actual model ID (e.g., "gpt-4")
						Object:  "model",
						Created: 1677657600, // Use a default timestamp for now
						OwnedBy: accessibleModel.Provider,
					})
				}
			}

			response := ModelsResponse{
				Object: "list",
				Data:   models,
			}

			log.Printf("Returning %d accessible models for authenticated user", len(models))
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// No authentication or fallback: return all models from database
	database, exists := c.Get("db")
	if !exists {
		log.Printf("Database connection not found in context")
		// Fallback to static models if no database
		c.JSON(http.StatusOK, getStaticModels())
		return
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		log.Printf("Invalid database connection type")
		// Fallback to static models if invalid database
		c.JSON(http.StatusOK, getStaticModels())
		return
	}

	// Get all models from database (for unauthenticated requests)
	dbModels, err := db.GetModelsWithOrganizations(sqlDB)
	if err != nil {
		log.Printf("Failed to get models from database: %v", err)
		// Fallback to static models if database error
		c.JSON(http.StatusOK, getStaticModels())
		return
	}

	// Convert database models to OpenAI-compatible format
	var models []Model
	for _, dbModel := range dbModels {
		if dbModel.IsActive {
			models = append(models, Model{
				ID:      dbModel.ModelID, // Use the actual model ID (e.g., "gpt-4")
				Object:  "model",
				Created: dbModel.CreatedAt.Unix(),
				OwnedBy: dbModel.Provider,
			})
		}
	}

	response := ModelsResponse{
		Object: "list",
		Data:   models,
	}

	log.Printf("Returning %d models for unauthenticated request", len(models))
	c.JSON(http.StatusOK, response)
}

// AccessibleModel represents a model that the organization has access to
// This should match the type in gateway/middleware/auth.go
type AccessibleModel struct {
	ID       string
	Name     string
	ModelID  string
	Provider string
	IsActive bool
}

// getStaticModels returns fallback static models when database is unavailable
func getStaticModels() ModelsResponse {
	return ModelsResponse{
		Object: "list",
		Data: []Model{
			{
				ID:      "gpt-3.5-turbo",
				Object:  "model",
				Created: 1677657600,
				OwnedBy: "openai",
			},
			{
				ID:      "gpt-4",
				Object:  "model",
				Created: 1677657600,
				OwnedBy: "openai",
			},
		},
	}
}
