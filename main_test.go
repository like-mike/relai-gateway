package main_test

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/like-mike/relai-gateway/provider"
// 	"github.com/like-mike/relai-gateway/routes"
// 	"github.com/sashabaranov/go-openai"
// )

// // Tests for provider.NewOpenAIProvider

// func TestNewOpenAIProvider_EmptyKey(t *testing.T) {
// 	_, err := provider.NewOpenAIProvider("", "")
// 	if err == nil {
// 		t.Fatalf("expected error for empty API key, got nil")
// 	}
// }

// func TestNewOpenAIProvider_DefaultModel(t *testing.T) {
// 	p, err := provider.NewOpenAIProvider("dummy-key", "")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if p.Model != openai.GPT3TextDavinci003 {
// 		t.Errorf("expected default model %q, got %q", openai.GPT3TextDavinci003, p.Model)
// 	}
// }

// func TestNewOpenAIProvider_CustomModel(t *testing.T) {
// 	p, err := provider.NewOpenAIProvider("dummy-key", "custom-model")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if p.Model != "custom-model" {
// 		t.Errorf("expected custom model %q, got %q", "custom-model", p.Model)
// 	}
// }

// // mockProvider implements provider.CompletionProvider for testing routes
// type mockProvider struct{}

// func (m *mockProvider) GetCompletions(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
// 	return &provider.CompletionResponse{Text: "mock-response"}, nil
// }

// // Tests for routes.HandleCompletionsHTTP

// func TestHandleCompletionsHTTP_MethodNotAllowed(t *testing.T) {
// 	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
// 	w := httptest.NewRecorder()
// 	routes.HandleCompletionsHTTP(w, req, &mockProvider{})
// 	if w.Code != http.StatusMethodNotAllowed {
// 		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
// 	}
// }

// func TestHandleCompletionsHTTP_InvalidJSON(t *testing.T) {
// 	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString("invalid-json"))
// 	w := httptest.NewRecorder()
// 	routes.HandleCompletionsHTTP(w, req, &mockProvider{})
// 	if w.Code != http.StatusBadRequest {
// 		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
// 	}
// }

// func TestHandleCompletionsHTTP_Success(t *testing.T) {
// 	body := routes.CompletionsRequest{Prompt: "hello", MaxTokens: 5}
// 	b, _ := json.Marshal(body)
// 	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(b))
// 	w := httptest.NewRecorder()
// 	routes.HandleCompletionsHTTP(w, req, &mockProvider{})
// 	if w.Code != http.StatusOK {
// 		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
// 	}
// 	var resp routes.CompletionsResponse
// 	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
// 		t.Fatalf("unexpected error decoding response: %v", err)
// 	}
// 	if resp.Text != "mock-response" {
// 		t.Errorf("expected response text %q, got %q", "mock-response", resp.Text)
// 	}
// }
