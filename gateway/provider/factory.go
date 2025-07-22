package provider

import (
	"github.com/like-mike/relai-gateway/shared/models"
)

// ProxyConfig holds configuration for making requests to AI providers
type ProxyConfig struct {
	BaseURL  string
	Name     string
	APIToken string
	ModelID  string
}

// CreateProxyConfigFromModel creates a ProxyConfig from a database model
func CreateProxyConfigFromModel(model *models.Model) *ProxyConfig {
	if model == nil {
		return getDefaultConfig()
	}

	baseURL := "https://api.openai.com" // default
	if model.APIEndpoint != nil && *model.APIEndpoint != "" {
		baseURL = *model.APIEndpoint
	}

	apiToken := ""
	if model.APIToken != nil {
		apiToken = *model.APIToken
	}

	return &ProxyConfig{
		BaseURL:  baseURL,
		Name:     model.Provider,
		APIToken: apiToken,
		ModelID:  model.ModelID,
	}
}

// getDefaultConfig provides a fallback configuration
func getDefaultConfig() *ProxyConfig {
	return &ProxyConfig{
		BaseURL:  "https://api.openai.com",
		Name:     "openai",
		APIToken: "",
		ModelID:  "gpt-3.5-turbo",
	}
}
