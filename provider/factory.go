package provider

import (
	"fmt"
	"os"
)

// NewProviderFromEnv constructs a CompletionProvider based on the LLM_PROVIDER environment variable.
// Supported values: "openai" (default if empty).
func NewProviderFromEnv() (CompletionProvider, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY environment variable is required")
	}

	providerName := os.Getenv("LLM_PROVIDER")
	switch providerName {
	case "", "openai":
		return NewOpenAIProvider(apiKey, os.Getenv("LLM_MODEL"))
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
