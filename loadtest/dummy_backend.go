package main

import (
	"encoding/json"
	"net/http"
)

type Choice struct {
	Text string `json:"text"`
}

type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

func completionsHandler(w http.ResponseWriter, r *http.Request) {
	resp := CompletionResponse{
		ID:      "dummy-id",
		Object:  "text_completion",
		Created: 0,
		Model:   "dummy-model",
		Choices: []Choice{{Text: "Hello from dummy backend!"}},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Only print errors, not 200s
		println("ERROR: failed to encode response:", err.Error())
	}
}

func main() {
	http.HandleFunc("/v1/chat/completions", completionsHandler)
	http.ListenAndServe(":2000", nil)
}
