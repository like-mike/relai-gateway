package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	providerpkg "github.com/like-mike/relai-gateway/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"status", "model", "user", "route"})

	httpRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"model", "user", "route"})

	httpErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"error", "model", "user", "route"})

	httpResponseCodesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_response_codes_total",
		Help: "Total number of HTTP response codes",
	}, []string{"code", "model", "user", "route"})

	llm_tokens = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llm_tokens",
		Help:    "Number of LLM tokens per completion",
		Buckets: prometheus.LinearBuckets(0, 50, 20),
	}, []string{"model", "user", "route"})
)

// ChatCompletionRequest represents the JSON body for chat completions.
type ChatCompletionRequest struct {
	Messages []providerpkg.ChatMessage `json:"messages"`
}

// CompletionsResponse represents the JSON response for chat completions.
type CompletionsResponse struct {
	Text string `json:"text"`
}

// RegisterRoutes registers the HTTP route for chat completions.
func RegisterRoutes(app *fiber.App, provider providerpkg.CompletionProvider) {
	app.Post("/v1/chat/completions", func(c *fiber.Ctx) error {
		start := time.Now()
		var req ChatCompletionRequest
		user := c.Get("X-User", "anonymous")
		model := ""
		route := "/v1/chat/completions"
		if mProvider, ok := provider.(interface{ Model() string }); ok {
			model = mProvider.Model()
		} else {
			model = "unknown"
		}

		if err := c.BodyParser(&req); err != nil {
			httpErrorsTotal.WithLabelValues("bad_request", model, user, route).Inc()
			httpResponseCodesTotal.WithLabelValues("400", model, user, route).Inc()
			httpRequestsTotal.WithLabelValues("bad_request", model, user, route).Inc()
			return c.Status(fiber.StatusBadRequest).SendString("invalid request body")
		}
		resp, err := provider.GetCompletions(c.Context(), &providerpkg.CompletionRequest{
			Messages: req.Messages,
		})
		latency := time.Since(start).Seconds()
		httpRequestDurationSeconds.WithLabelValues(model, user, route).Observe(latency)
		if err != nil {
			httpErrorsTotal.WithLabelValues("provider_error", model, user, route).Inc()
			httpResponseCodesTotal.WithLabelValues("500", model, user, route).Inc()
			httpRequestsTotal.WithLabelValues("error", model, user, route).Inc()
			return c.Status(fiber.StatusInternalServerError).SendString("error fetching chat completion: " + err.Error())
		}
		// Token counting (simple heuristic: word count)
		tokenCount := float64(len(resp.Text) / 4)
		llm_tokens.WithLabelValues(model, user, route).Observe(tokenCount)
		httpResponseCodesTotal.WithLabelValues("200", model, user, route).Inc()
		httpRequestsTotal.WithLabelValues("success", model, user, route).Inc()
		return c.JSON(CompletionsResponse{Text: resp.Text})
	})
}

// HandleCompletionsHTTP is an HTTP handler used in tests for chat completions.
func HandleCompletionsHTTP(w http.ResponseWriter, r *http.Request, provider providerpkg.CompletionProvider) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := provider.GetCompletions(r.Context(), &providerpkg.CompletionRequest{
		Messages: req.Messages,
	})
	if err != nil {
		http.Error(w, "error fetching chat completion: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CompletionsResponse{Text: resp.Text})
}
