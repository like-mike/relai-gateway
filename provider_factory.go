package main

import (
	"fmt"
	"os"

	"github.com/like-mike/relai-gateway/provider"
)

// NewProviderFromEnv constructs a CompletionProvider based on the LLM_PROVIDER environment variable.
// Supported values: "openai" (default if empty).
// Uses OPENAI_API_KEY and OPENAI_MODEL for OpenAI provider configuration.
func NewProviderFromEnv() (provider.CompletionProvider, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY environment variable is required")
	}

	providerName := os.Getenv("LLM_PROVIDER")
	switch providerName {
	case "", "openai":
		return provider.NewOpenAIProvider(apiKey, os.Getenv("LLM_MODEL"))
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
