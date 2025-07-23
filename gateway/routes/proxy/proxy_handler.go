// routes/proxy.go
package proxy

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/gateway/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// createHTTPClientForModel creates an HTTP client with model-specific timeout
func createHTTPClientForModel(cfg *middleware.AccessibleModel) *http.Client {
	timeout := 30 * time.Second // default timeout
	if cfg.ModelID != "" && cfg.TimeoutSeconds != nil {
		timeout = time.Duration(*cfg.TimeoutSeconds) * time.Second
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}
}

// makeRequestWithRetry executes HTTP request with model-specific retry logic
func makeRequestWithRetry(client *http.Client, req *http.Request, bodyBytes []byte, cfg *middleware.AccessibleModel) (*http.Response, error) {
	// Default retry settings
	maxRetries := 2
	retryDelay := 1000 * time.Millisecond
	backoffMultiplier := 2.0

	// Use model-specific settings if available
	if cfg.ID != "" {
		if cfg.MaxRetries != nil {
			maxRetries = *cfg.MaxRetries
		}
		if cfg.RetryDelayMs != nil {
			retryDelay = time.Duration(*cfg.RetryDelayMs) * time.Millisecond
		}
		if cfg.BackoffMultiplier != nil {
			backoffMultiplier = *cfg.BackoffMultiplier
		}
	}

	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(retryDelay) * math.Pow(backoffMultiplier, float64(attempt-1)))
			log.Printf("Retrying request to %s (attempt %d/%d) after %v", req.URL.Host, attempt+1, maxRetries+1, delay)
			time.Sleep(delay)
		}

		// Create fresh request with body for each attempt
		reqClone, err := http.NewRequest(req.Method, req.URL.String(), strings.NewReader(string(bodyBytes)))
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %v", err)
		}

		// Copy headers from original request
		for k, v := range req.Header {
			reqClone.Header[k] = v
		}

		// Copy context
		reqClone = reqClone.WithContext(req.Context())

		resp, err := client.Do(reqClone)
		if err == nil {
			// Check if response indicates success or retryable error
			if resp.StatusCode < 500 {
				// Success or client error (don't retry 4xx)
				return resp, nil
			}
			// Server error (5xx) - close body and retry
			if lastResp != nil {
				lastResp.Body.Close()
			}
			lastResp = resp
		}

		lastErr = err
		log.Printf("Request failed on attempt %d: status=%d, error=%v", attempt+1,
			func() int {
				if resp != nil {
					return resp.StatusCode
				}
				return 0
			}(), err)
	}

	// All retries exhausted
	if lastResp != nil {
		// Return the last response even if it's an error
		return lastResp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %v", maxRetries+1, lastErr)
}

func Handler(c *gin.Context) {
	ctx := c.Request.Context()
	tracer := otel.GetTracerProvider().Tracer("gateway")
	fmt.Println("Proxy handler invoked")

	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery
	target := path
	if query != "" {
		target += "?" + query
	}

	// Check for custom endpoints first
	// customEndpoint := checkForCustomEndpoint(c, path)
	// var model *models.Model
	// var cfg *provider.ProxyConfig

	// if customEndpoint != nil {
	// 	log.Printf("Using custom endpoint: %s for path: %s", customEndpoint.Name, path)
	// 	// Get the model from database
	// 	if customEndpoint.PrimaryModelID != nil {
	// 		model = getModelByID(c, *customEndpoint.PrimaryModelID)
	// 	}
	// 	// Update the target path to remove the custom prefix and use standard API paths
	// 	target = convertCustomPathToStandard(path, customEndpoint.PathPrefix, target)
	// } else {
	// 	// For non-custom endpoints, we could look up model by other means
	// 	// For now, use default - this could be enhanced to parse model from request
	// 	model = nil
	// }

	// Create provider config from model (or use default if no model)

	// cfg = provider.CreateProxyConfigFromModel(model)

	// Build proxy request
	cfg, req, bodyBytes, err := prepareRequest(c, target)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Trace the provider call
	ctx, spanInvoke := tracer.Start(ctx, "invoke_provider")
	defer spanInvoke.End()

	ctx, spanExec := tracer.Start(ctx, cfg.Name)
	defer spanExec.End()

	recordTracingMetadata(cfg, spanInvoke, spanExec, req, bodyBytes)

	// Send request with model-specific retry/timeout
	start := time.Now()

	client := createHTTPClientForModel(cfg)

	// Execute request with retry logic
	resp, err := makeRequestWithRetry(client, req, bodyBytes, cfg)

	duration := time.Since(start).Milliseconds()
	spanInvoke.SetAttributes(attribute.Int64("llm.request.duration_ms", duration))

	// Build response
	writeDownstreamResponse(cfg, c, resp, err, tracer, start)
}

// CustomEndpoint represents a custom endpoint from the database
type CustomEndpoint struct {
	ID              string
	OrganizationID  string
	Name            string
	PathPrefix      string
	Description     string
	PrimaryModelID  *string
	FallbackModelID *string
	IsActive        bool
}

// checkForCustomEndpoint checks if the current path matches a custom endpoint
func checkForCustomEndpoint(c *gin.Context, path string) *CustomEndpoint {
	// Get database connection from context
	database, exists := c.Get("db")
	if !exists {
		return nil
	}

	sqlDB, ok := database.(*sql.DB)
	if !ok {
		return nil
	}

	// Get organization ID from API key context (if authenticated)
	orgID, exists := c.Get("organization_id")
	if !exists {
		// If no organization context, check for public endpoints
		// For now, return nil - could be enhanced to support public endpoints
		return nil
	}

	orgIDStr, ok := orgID.(string)
	if !ok {
		return nil
	}

	// Extract the potential custom path prefix from the URL
	// Expected format: /api/{custom_prefix}/...
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "api" {
		return nil
	}

	customPrefix := pathParts[1]

	// Query for matching custom endpoint
	query := `
		SELECT id, organization_id, name, path_prefix, description, primary_model_id, fallback_model_id, is_active
		FROM endpoints
		WHERE organization_id = $1 AND path_prefix = $2 AND is_active = true
	`

	var endpoint CustomEndpoint
	err := sqlDB.QueryRow(query, orgIDStr, customPrefix).Scan(
		&endpoint.ID,
		&endpoint.OrganizationID,
		&endpoint.Name,
		&endpoint.PathPrefix,
		&endpoint.Description,
		&endpoint.PrimaryModelID,
		&endpoint.FallbackModelID,
		&endpoint.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No custom endpoint found, continue with default behavior
			return nil
		}
		log.Printf("Error querying custom endpoint: %v", err)
		return nil
	}

	return &endpoint
}

// getProviderConfigForModel is now replaced by provider.CreateProxyConfigFromModel
// which uses the full model data from the database instead of hardcoded configs

// convertCustomPathToStandard converts custom endpoint paths to standard API paths
// func convertCustomPathToStandard(originalPath, customPrefix, target string) string {
// 	// Remove the custom prefix and convert to standard OpenAI API path
// 	// Example: /api/chat/completions -> /v1/chat/completions
// 	// Example: /api/custom-assistant/completions -> /v1/chat/completions

// 	standardPath := strings.Replace(originalPath, "/api/"+customPrefix, "/v1", 1)

// 	// If the path doesn't have a specific endpoint, default to chat/completions
// 	if standardPath == "/v1" || standardPath == "/v1/" {
// 		standardPath = "/v1/chat/completions"
// 	}

// 	// Update the target with the new path
// 	if strings.Contains(target, "?") {
// 		parts := strings.Split(target, "?")
// 		return standardPath + "?" + parts[1]
// 	}

// 	return standardPath
// }

// getModelByID retrieves a model from the database by ID
// func getModelByID(c *gin.Context, modelID string) *models.Model {
// 	database, exists := c.Get("db")
// 	if !exists {
// 		return nil
// 	}

// 	sqlDB, ok := database.(*sql.DB)
// 	if !ok {
// 		return nil
// 	}

// 	query := `SELECT id, name, description, provider, model_id, api_endpoint, api_token,
// 	          input_cost_per_1m, output_cost_per_1m, max_retries, timeout_seconds,
// 	          retry_delay_ms, backoff_multiplier, is_active, created_at, updated_at
// 			  FROM models WHERE id = $1 AND is_active = true`

// 	var model models.Model
// 	err := sqlDB.QueryRow(query, modelID).Scan(
// 		&model.ID, &model.Name, &model.Description, &model.Provider,
// 		&model.ModelID, &model.APIEndpoint, &model.APIToken,
// 		&model.InputCostPer1M, &model.OutputCostPer1M,
// 		&model.MaxRetries, &model.TimeoutSeconds, &model.RetryDelayMs, &model.BackoffMultiplier,
// 		&model.IsActive, &model.CreatedAt, &model.UpdatedAt,
// 	)
// 	if err != nil {
// 		log.Printf("Error getting model %s: %v", modelID, err)
// 		return nil
// 	}

// 	return &model
// }
