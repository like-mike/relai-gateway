package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/provider"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func prepareRequest(cfg *provider.ProxyConfig, c *gin.Context, target string) (*http.Request, []byte, error) {
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	dummyBackend := os.Getenv("USE_DUMMY_BACKEND")
	var baseURL string
	if dummyBackend == "1" {
		log.Println("Using dummy backend for testing")
		baseURL = "http://dummy-backend:2000"
	} else {
		// config := provider.NewProxyConfigFromEnv("openai")
		baseURL = cfg.BaseURL
	}

	req, err := http.NewRequest(c.Request.Method, baseURL+target, io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		return nil, nil, err
	}

	for k, v := range c.Request.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	// body, _ := io.ReadAll(c.Request.Body)

	// modelName, err := DetectModel(body)
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("failed to detect model: %w", err)
	// }

	modelName := "gpt-4.1"

	model := provider.ModelMap[modelName]

	if dummyBackend != "1" {
		req.Header.Set("Authorization", "Bearer "+model.SecretKey)
	}

	return req, bodyBytes, nil
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

func writeDownstreamResponse(c *gin.Context, resp *http.Response, err error, tracer trace.Tracer) {
	_, span := tracer.Start(c.Request.Context(), "build_response")
	defer span.End()

	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", err.Error()),
			attribute.Int("http.status_code", http.StatusBadGateway),
		)
		c.String(http.StatusBadGateway, "failed to reach provider")
		return
	}
	defer resp.Body.Close()

	for hk, hv := range resp.Header {
		for _, v := range hv {
			if hk != "Set-Cookie" {
				c.Writer.Header().Add(hk, v)
				// fmt.Println(hk, v)
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

	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		span.SetAttributes(attribute.String("error.message", err.Error()))
		c.String(http.StatusInternalServerError, "failed to stream provider response")
		return
	}
}
