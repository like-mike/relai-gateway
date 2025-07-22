package usage

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/like-mike/relai-gateway/shared/models"
)

// CostCalculator calculates costs for AI provider usage
type CostCalculator interface {
	CalculateCost(usage *models.AIProviderUsage, modelID string) (float64, error)
	GetProviderName() string
}

// DatabaseCostCalculator uses database-stored pricing
type DatabaseCostCalculator struct {
	database *sql.DB
	provider string
}

func NewDatabaseCostCalculator(database *sql.DB, provider string) *DatabaseCostCalculator {
	return &DatabaseCostCalculator{
		database: database,
		provider: provider,
	}
}

func (c *DatabaseCostCalculator) GetProviderName() string {
	return c.provider
}

func (c *DatabaseCostCalculator) CalculateCost(usage *models.AIProviderUsage, modelID string) (float64, error) {
	// If no database available, use fallback immediately
	if c.database == nil {
		return c.calculateFallbackCost(usage, modelID)
	}

	// Get model with cost information
	model, err := c.getModelCostData(modelID)
	if err != nil {
		// Fall back to generic pricing if database lookup fails
		log.Printf("Failed to get model from database for %s, using fallback: %v", modelID, err)
		return c.calculateFallbackCost(usage, modelID)
	}

	// Use model's cost fields if available
	if model.InputCostPer1M != nil && model.OutputCostPer1M != nil &&
		*model.InputCostPer1M > 0 && *model.OutputCostPer1M > 0 {
		// Calculate cost using model pricing (convert from per-1M to per-1K)
		inputCostPer1K := *model.InputCostPer1M / 1000.0
		outputCostPer1K := *model.OutputCostPer1M / 1000.0

		inputCost := float64(usage.PromptTokens) / 1000.0 * inputCostPer1K
		outputCost := float64(usage.CompletionTokens) / 1000.0 * outputCostPer1K

		totalCost := inputCost + outputCost

		log.Printf("Calculated cost for model %s using stored pricing: $%.6f (input: $%.6f, output: $%.6f)",
			modelID, totalCost, inputCost, outputCost)
		return totalCost, nil
	}

	// Fall back to generic pricing if no cost configured
	log.Printf("No cost configuration found for model %s, using fallback pricing", modelID)
	return c.calculateFallbackCost(usage, modelID)
}

func (c *DatabaseCostCalculator) getModelCostData(modelID string) (*models.Model, error) {
	if c.database == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	query := `
		SELECT id, name, description, provider, model_id, api_endpoint, api_token,
		       input_cost_per_1m, output_cost_per_1m, is_active, created_at, updated_at
		FROM models
		WHERE id = $1
	`

	var model models.Model
	err := c.database.QueryRow(query, modelID).Scan(
		&model.ID, &model.Name, &model.Description, &model.Provider,
		&model.ModelID, &model.APIEndpoint, &model.APIToken,
		&model.InputCostPer1M, &model.OutputCostPer1M,
		&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &model, nil
}

func (c *DatabaseCostCalculator) calculateFallbackCost(usage *models.AIProviderUsage, modelID string) (float64, error) {
	// Simple fallback: use generic pricing estimate
	// $0.002 per 1K tokens (similar to GPT-3.5 pricing)
	pricePerThousandTokens := 0.002
	totalCost := (float64(usage.TotalTokens) / 1000.0) * pricePerThousandTokens

	log.Printf("Using fallback cost calculation for provider '%s', model '%s': $%.6f",
		c.provider, modelID, totalCost)

	return totalCost, nil
}

// CostCalculatorFactory creates the appropriate cost calculator for a provider
type CostCalculatorFactory struct {
	database *sql.DB
}

func NewCostCalculatorFactory() *CostCalculatorFactory {
	return &CostCalculatorFactory{}
}

func NewCostCalculatorFactoryWithDB(database *sql.DB) *CostCalculatorFactory {
	return &CostCalculatorFactory{database: database}
}

func (f *CostCalculatorFactory) GetCalculator(provider string) CostCalculator {
	if f.database != nil {
		return NewDatabaseCostCalculator(f.database, provider)
	}

	return NewDatabaseCostCalculator(nil, provider)
}

// CalculateCostForUsage calculates the cost for usage data from a specific provider
// This function is kept for backward compatibility but now uses database-driven pricing
func CalculateCostForUsage(usage *models.AIProviderUsage, provider, modelID string) (float64, error) {
	if usage == nil {
		return 0, fmt.Errorf("usage data is nil")
	}

	factory := NewCostCalculatorFactory()
	calculator := factory.GetCalculator(provider)
	return calculator.CalculateCost(usage, modelID)
}
