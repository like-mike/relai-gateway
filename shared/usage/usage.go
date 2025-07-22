package usage

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/like-mike/relai-gateway/shared/models"
)

// UsageTracker provides a high-level interface for usage tracking
type UsageTracker struct {
	workerPool        *UsageWorkerPool
	extractorFactory  *ExtractorFactory
	calculatorFactory *CostCalculatorFactory
	enabled           bool
}

// NewUsageTracker creates a new usage tracker instance
func NewUsageTracker(database *sql.DB, config *WorkerConfig) *UsageTracker {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	workerPool := NewUsageWorkerPool(database, config)
	workerPool.Start()

	return &UsageTracker{
		workerPool:        workerPool,
		extractorFactory:  NewExtractorFactory(),
		calculatorFactory: NewCostCalculatorFactoryWithDB(database),
		enabled:           true,
	}
}

// SetEnabled enables or disables usage tracking
func (t *UsageTracker) SetEnabled(enabled bool) {
	t.enabled = enabled
	log.Printf("Usage tracking %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// IsEnabled returns whether usage tracking is enabled
func (t *UsageTracker) IsEnabled() bool {
	return t.enabled
}

// TrackUsage extracts usage from response and submits it for logging
func (t *UsageTracker) TrackUsage(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte,
) {
	if !t.enabled {
		return
	}

	// Process in background to avoid blocking the response
	go func() {
		if err := t.processUsage(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS, responseBody,
		); err != nil {
			// If standard extraction failed, check if we can use tiktoken
			// This handles streaming responses automatically
			if strings.Contains(err.Error(), "streaming response detected") {
				log.Printf("Streaming response detected, attempting tiktoken extraction...")
				// We need the request body for tiktoken, but we don't have it here
				// The proxy layer should handle this with TrackUsageWithTiktoken instead
				log.Printf("Use tiktoken extraction at proxy layer for accurate streaming tracking")
			} else if !strings.Contains(err.Error(), "not valid JSON") &&
				!strings.Contains(err.Error(), "no usage data found") {
				log.Printf("Failed to process usage tracking: %v", err)
			}
		}
	}()
}

// processUsage handles the actual usage extraction and cost calculation
func (t *UsageTracker) processUsage(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte,
) error {
	// Extract usage from response
	extractor := t.extractorFactory.GetExtractor(provider)
	usage, err := extractor.ExtractUsage(responseBody)
	if err != nil {
		// If normal extraction failed, it might be a streaming response
		// Log the failure and let caller handle tiktoken fallback
		log.Printf("Standard usage extraction failed for %s: %v", provider, err)
		return err
	}

	// Calculate cost
	calculator := t.calculatorFactory.GetCalculator(provider)
	cost, err := calculator.CalculateCost(usage, modelID)
	if err != nil {
		log.Printf("Failed to calculate cost for provider %s, model %s: %v", provider, modelID, err)
		// Continue without cost if calculation fails
		cost = 0
	}

	// Prepare metadata
	metadata := map[string]interface{}{
		"provider":        provider,
		"model_id":        modelID,
		"extraction_type": "standard",
		"extracted_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Submit to worker pool
	success := t.workerPool.SubmitUsage(
		orgID, apiKeyID, modelID, provider, endpoint,
		requestID, responseStatus, responseTimeMS,
		usage, &cost, metadata,
	)

	if !success {
		log.Printf("Failed to submit usage job to worker pool (queue full)")
		return err
	}

	log.Printf("Successfully queued standard usage tracking for org %s: %d tokens, $%.6f",
		orgID, usage.TotalTokens, cost)

	return nil
}

// TrackUsageWithData allows manual submission of usage data
func (t *UsageTracker) TrackUsageWithData(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	usage *models.AIProviderUsage,
) {
	if !t.enabled || usage == nil {
		return
	}

	// Process in background
	go func() {
		// Calculate cost
		calculator := t.calculatorFactory.GetCalculator(provider)
		cost, err := calculator.CalculateCost(usage, modelID)
		if err != nil {
			log.Printf("Failed to calculate cost for provider %s, model %s: %v", provider, modelID, err)
			cost = 0
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"provider":          provider,
			"model_id":          modelID,
			"manual_submission": true,
			"submitted_at":      time.Now().UTC().Format(time.RFC3339),
		}

		// Submit to worker pool
		success := t.workerPool.SubmitUsage(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS,
			usage, &cost, metadata,
		)

		if !success {
			log.Printf("Failed to submit manual usage job to worker pool (queue full)")
		}
	}()
}

// TrackUsageWithTiktoken uses tiktoken for accurate streaming response tracking
func (t *UsageTracker) TrackUsageWithTiktoken(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte, requestBody []byte,
) {
	if !t.enabled {
		return
	}

	// Process in background
	go func() {
		// Use tiktoken extractor for accurate token counting
		extractor := NewTiktokenExtractor(modelID)
		usage, err := extractor.ExtractFromStreamingResponse(responseBody, requestBody)
		if err != nil {
			log.Printf("Tiktoken extraction failed, falling back to normal extraction: %v", err)
			// Fall back to normal processing
			if err := t.processUsage(
				orgID, apiKeyID, modelID, provider, endpoint,
				requestID, responseStatus, responseTimeMS, responseBody,
			); err != nil {
				log.Printf("Both tiktoken and normal extraction failed: %v", err)
			}
			return
		}

		// Calculate cost
		calculator := t.calculatorFactory.GetCalculator(provider)
		cost, err := calculator.CalculateCost(usage, modelID)
		if err != nil {
			log.Printf("Failed to calculate cost for provider %s, model %s: %v", provider, modelID, err)
			cost = 0
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"provider":     provider,
			"model_id":     modelID,
			"tiktoken":     true,
			"extracted_at": time.Now().UTC().Format(time.RFC3339),
		}

		// Submit to worker pool
		success := t.workerPool.SubmitUsage(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS,
			usage, &cost, metadata,
		)

		if !success {
			log.Printf("Failed to submit tiktoken usage job to worker pool (queue full)")
			return
		}

		log.Printf("Successfully tracked usage with tiktoken for org %s: %d tokens, $%.6f",
			orgID, usage.TotalTokens, cost)
	}()
}

// Stop gracefully shuts down the usage tracker
func (t *UsageTracker) Stop() {
	log.Println("Stopping usage tracker...")
	t.workerPool.Stop()
}

// GetStats returns statistics about the usage tracking system
func (t *UsageTracker) GetStats() UsageTrackerStats {
	workerStats := t.workerPool.GetStats()

	return UsageTrackerStats{
		Enabled:         t.enabled,
		WorkerPoolStats: workerStats,
	}
}

// UsageTrackerStats contains statistics about the usage tracker
type UsageTrackerStats struct {
	Enabled         bool            `json:"enabled"`
	WorkerPoolStats WorkerPoolStats `json:"worker_pool_stats"`
}

// Global usage tracker instance
var globalUsageTracker *UsageTracker

// InitGlobalUsageTracker initializes the global usage tracker
func InitGlobalUsageTracker(database *sql.DB, config *WorkerConfig) {
	if globalUsageTracker != nil {
		log.Println("Global usage tracker already initialized")
		return
	}

	globalUsageTracker = NewUsageTracker(database, config)
	log.Println("Global usage tracker initialized")
}

// GetGlobalUsageTracker returns the global usage tracker instance
func GetGlobalUsageTracker() *UsageTracker {
	return globalUsageTracker
}

// StopGlobalUsageTracker stops the global usage tracker
func StopGlobalUsageTracker() {
	if globalUsageTracker != nil {
		globalUsageTracker.Stop()
		globalUsageTracker = nil
	}
}

// Convenience functions for global usage tracker

// TrackUsage is a convenience function to track usage with the global tracker
func TrackUsage(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte,
) {
	if globalUsageTracker != nil {
		globalUsageTracker.TrackUsage(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS, responseBody,
		)
	}
}

// TrackUsageWithData is a convenience function to track usage data with the global tracker
func TrackUsageWithData(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	usage *models.AIProviderUsage,
) {
	if globalUsageTracker != nil {
		globalUsageTracker.TrackUsageWithData(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS, usage,
		)
	}
}

// TrackUsageWithTiktoken is a convenience function to track usage with tiktoken with the global tracker
func TrackUsageWithTiktoken(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte, requestBody []byte,
) {
	if globalUsageTracker != nil {
		globalUsageTracker.TrackUsageWithTiktoken(
			orgID, apiKeyID, modelID, provider, endpoint,
			requestID, responseStatus, responseTimeMS, responseBody, requestBody,
		)
	}
}
