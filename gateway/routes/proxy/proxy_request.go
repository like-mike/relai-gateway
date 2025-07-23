package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/gateway/middleware"
	"github.com/like-mike/relai-gateway/shared/usage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func prepareRequest(c *gin.Context, target string) (*middleware.AccessibleModel, *http.Request, []byte, error) {
	var cfg *middleware.AccessibleModel

	bodyBytes, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 1. Detect the model requested in the body
	modelName, err := DetectModel(bodyBytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to detect model: %w", err)
	}

	fmt.Println("Did you get this far? Model detected:", modelName)

	// 2. Get accessible models from auth middleware context
	accessibleModelsInterface, exists := c.Get("accessible_models")
	if !exists {
		return nil, nil, nil, fmt.Errorf("no accessible models found in context - authentication required")
	}

	accessibleModels, ok := accessibleModelsInterface.([]middleware.AccessibleModel)
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid accessible models format in context")
	}

	// 3. Check if organization has access to the requested model and get its API token
	// var modelApiToken string
	// var accessibleModelID string
	var hasAccess bool
	for _, accessibleModel := range accessibleModels {

		if accessibleModel.ModelID == modelName {
			cfg = &accessibleModel // Use the current model in the loop
			hasAccess = true
			// modelApiToken = accessibleModel.ApiToken
			// accessibleModelID = accessibleModel.ID
			log.Printf("Organization has access to model %s (provider: %s)", modelName, accessibleModel.Provider)
			break
		}
	}

	log.Println("cfg", cfg)

	if !hasAccess {
		return nil, nil, nil, fmt.Errorf("organization does not have access to model: %s", modelName)
	}

	// Store model ID in context for usage logging
	c.Set("model_id", cfg.ModelID)

	// Store request body for tokenizer fallback in streaming responses
	c.Set("request_body", bodyBytes)

	// Get organization ID for logging
	organizationID, _ := c.Get("organization_id")
	log.Printf("Request authenticated - Model: %s, Organization: %v", modelName, organizationID)

	// 4. Prepare the upstream request
	dummyBackend := os.Getenv("USE_DUMMY_BACKEND")
	var baseURL string
	if dummyBackend == "1" {
		log.Println("Using dummy backend for testing")
		baseURL = os.Getenv("DUMMY_BACKEND_HOST")
		if baseURL == "" {
			return nil, nil, nil, fmt.Errorf("DUMMY_BACKEND_HOST environment variable is not set")
		}
	} else {
		baseURL = cfg.ApiEndpoint
	}

	// TODO: something here for when users enter /v1 in the ui, route already captures everything after host
	log.Println("URL for model:", baseURL+target)
	req, err := http.NewRequest(c.Request.Method, baseURL+target, io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		return nil, nil, nil, err
	}

	// Copy headers from original request
	for k, v := range c.Request.Header {
		for _, vv := range v {
			if k != "Authorization" {
				req.Header.Add(k, vv)
			}

		}
	}

	// 5. Set the correct API token for the model (not dummy backend)
	if dummyBackend != "1" {
		req.Header.Set("Authorization", "Bearer "+cfg.ApiToken)
		log.Printf("Using model-specific API token for %s", modelName)
	}

	return cfg, req, bodyBytes, nil
}

type ChatCompletionRequest struct {
	Model string `json:"model"`
}

// DetectModel parses the JSON and returns the value of the "model" field
func DetectModel(jsonInput []byte) (string, error) {
	var req ChatCompletionRequest
	err := json.Unmarshal(jsonInput, &req)
	if err != nil {
		return "", err
	}
	return req.Model, nil
}

func writeDownstreamResponse(cfg *middleware.AccessibleModel, c *gin.Context, resp *http.Response, err error, tracer trace.Tracer, startTime time.Time) {
	_, span := tracer.Start(c.Request.Context(), "build_response")
	defer span.End()

	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", err.Error()),
			attribute.Int("http.status_code", http.StatusBadGateway),
		)
		c.String(http.StatusBadGateway, "failed to reach provider")

		// Track the failed request
		c.Status(http.StatusBadGateway)
		errorResponse := []byte(`{"error": {"message": "failed to reach provider", "type": "gateway_error"}}`)
		trackUsageFromResponse(cfg, c, errorResponse, startTime)
		return
	}
	defer resp.Body.Close()

	// Copy headers to client
	for hk, hv := range resp.Header {
		for _, v := range hv {
			if hk != "Set-Cookie" {
				c.Writer.Header().Add(hk, v)
			}
		}
	}

	c.Status(resp.StatusCode)
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("http.status_reason", http.StatusText(resp.StatusCode)),
	)

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.String("error.message", http.StatusText(resp.StatusCode)))
	}

	// Check if this is a streaming response by looking at Content-Type header
	contentType := resp.Header.Get("Content-Type")
	isStreamingResponse := strings.Contains(contentType, "text/event-stream") || strings.Contains(contentType, "text/plain")

	if isStreamingResponse {
		log.Printf("Detected streaming response, using optimized streaming with flushing")
		// For streaming responses, use chunk-by-chunk reading with explicit flushing
		var responseBuffer bytes.Buffer
		buffer := make([]byte, 4096) // Optimized buffer size

		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				// Write to client immediately
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					span.SetAttributes(attribute.String("error.message", writeErr.Error()))
					log.Printf("Failed to write streaming chunk: %v", writeErr)
					return
				}

				// Flush immediately for real-time delivery
				if flusher, ok := c.Writer.(http.Flusher); ok {
					flusher.Flush()
				}

				// Also capture for token logging (efficient in-memory operation)
				responseBuffer.Write(buffer[:n])
			}

			if err != nil {
				if err == io.EOF {
					log.Printf("Streaming completed successfully")
					break
				}
				span.SetAttributes(attribute.String("error.message", err.Error()))
				log.Printf("Error reading streaming response: %v", err)
				break
			}
		}

		// Track usage with captured response data
		responseBody := responseBuffer.Bytes()
		log.Printf("Streaming response completed - Length: %d", len(responseBody))
		trackUsageFromResponse(cfg, c, responseBody, startTime)
	} else {
		log.Printf("Detected non-streaming response, reading full body")
		// For non-streaming responses, read all then write (existing behavior)
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			span.SetAttributes(attribute.String("error.message", err.Error()))
			c.String(http.StatusInternalServerError, "failed to read provider response")

			// Track the failed request
			c.Status(http.StatusInternalServerError)
			errorResponse := []byte(`{"error": {"message": "failed to read provider response", "type": "gateway_error"}}`)
			trackUsageFromResponse(cfg, c, errorResponse, startTime)
			return
		}

		// Write response body to client
		if _, err = c.Writer.Write(responseBody); err != nil {
			span.SetAttributes(attribute.String("error.message", err.Error()))
			c.String(http.StatusInternalServerError, "failed to write provider response")
			return
		}

		log.Printf("Non-streaming response completed - Length: %d", len(responseBody))
		trackUsageFromResponse(cfg, c, responseBody, startTime)
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// trackUsageFromResponse extracts and tracks usage from the provider response
func trackUsageFromResponse(cfg *middleware.AccessibleModel, c *gin.Context, responseBody []byte, startTime time.Time) {
	// Get context data for usage tracking
	orgID, _ := c.Get("organization_id")
	apiKeyID, _ := c.Get("api_key_id")

	// apiKeyID = cfg.
	modelIDStr := cfg.ID

	orgIDStr, _ := orgID.(string)
	apiKeyIDStr, _ := apiKeyID.(string)
	// apiKeyIDStr := "empty"
	// modelIDStr, ok3 := modelID.(string)

	// if !ok1 || !ok3 {
	// 	log.Printf("Missing required context for usage tracking: org=%v, apiKey=%v, model=%v", ok1, ok2, ok3)
	// 	return
	// }
	log.Println("Tracking usage for org:", orgIDStr, "apiKey:", apiKeyIDStr, "model:", modelIDStr)

	// Determine provider from accessible models
	provider := "unknown"
	accessibleModelsInterface, exists := c.Get("accessible_models")
	if exists {
		if accessibleModels, ok := accessibleModelsInterface.([]middleware.AccessibleModel); ok {
			for _, model := range accessibleModels {
				// log.Println("ID:", model.ID, "ModelID:", modelIDStr)
				if model.ID == modelIDStr {
					provider = model.Provider
					break
				}
			}
		}
	}

	// Calculate response time
	responseTimeMS := int(time.Since(startTime).Milliseconds())

	// Get endpoint from request
	endpoint := c.Request.URL.Path

	// Extract request ID from response headers (if available)
	var requestID *string
	if reqID := c.Writer.Header().Get("X-Request-Id"); reqID != "" {
		requestID = &reqID
	}

	// Check if this is a streaming response - use tiktoken for all streaming
	isStreaming := len(responseBody) > 0 && strings.Contains(string(responseBody[:min(100, len(responseBody))]), "data:")

	if isStreaming {
		// Use tiktoken for streaming responses
		if requestBody, exists := c.Get("request_body"); exists {
			if requestBodyBytes, ok := requestBody.([]byte); ok {
				log.Printf("Using tiktoken for streaming response (model: %s)", modelIDStr)
				trackUsageWithTokenizer(
					orgIDStr, apiKeyIDStr, modelIDStr, provider, endpoint,
					requestID, c.Writer.Status(), &responseTimeMS,
					responseBody, requestBodyBytes,
				)
				return
			}
		}
		log.Printf("Streaming detected but no request body available for tiktoken")
	}

	// Use standard tracking for non-streaming responses
	usage.TrackUsage(
		orgIDStr,
		apiKeyIDStr,
		modelIDStr,
		provider,
		endpoint,
		requestID,
		c.Writer.Status(),
		&responseTimeMS,
		responseBody,
	)
}

// trackUsageWithTokenizer uses tiktoken for accurate streaming response tracking
func trackUsageWithTokenizer(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	responseBody []byte, requestBody []byte,
) {
	// Use tiktoken for accurate token counting
	usage.TrackUsageWithTiktoken(
		orgID, apiKeyID, modelID, provider, endpoint,
		requestID, responseStatus, responseTimeMS,
		responseBody, requestBody,
	)
}
