package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/like-mike/relai-gateway/metrics"
	providerpkg "github.com/like-mike/relai-gateway/provider"
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
		var req ChatCompletionRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("invalid request body")
		}
		resp, err := provider.GetCompletions(c.Context(), &providerpkg.CompletionRequest{
			Messages: req.Messages,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("error fetching chat completion: " + err.Error())
		}
		// Token counting (simple heuristic: word count)
		tokenCount := float64(len(resp.Text) / 4)
		metrics.LlmTokens.WithLabelValues(c.OriginalURL()).Observe(tokenCount)
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
