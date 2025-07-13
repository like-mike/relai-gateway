package provider

import (
	"os"
)

type ProxyConfig struct {
	BaseURL    string
	AuthHeader string
}

func NewProxyConfigFromEnv(providerOverride string) *ProxyConfig {
	provider := providerOverride
	if provider == "" {
		provider = os.Getenv("LLM_PROVIDER")
	}
	switch provider {
	case "aws":
		return &ProxyConfig{
			BaseURL:    "https://api.aws.com",
			AuthHeader: "Authorization",
		}
	case "ollama":
		return &ProxyConfig{
			BaseURL:    "http://localhost:11434",
			AuthHeader: "",
		}
	default: // openai
		return &ProxyConfig{
			BaseURL:    "https://api.openai.com",
			AuthHeader: "Authorization",
		}
	}
}
