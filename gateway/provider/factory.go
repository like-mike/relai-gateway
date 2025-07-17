package provider

import (
	"os"

	"github.com/joho/godotenv"
)

type Model struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int64  `json:"created"`
	OwnedBy   string `json:"owned_by"`
	SecretKey string
}

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

var MODELS ModelList

var ModelMap map[string]Model

type ProxyConfig struct {
	BaseURL string
	Name    string
}

func init() {

	//temp
	_ = godotenv.Load()

	ModelMap = make(map[string]Model)
	MODELS = ModelList{
		Object: "list",
		Data: []Model{
			{
				ID:        "gpt-3.5-turbo",
				Object:    "model",
				Created:   1677657600,
				OwnedBy:   "openai",
				SecretKey: os.Getenv("LLM_API_KEY"),
			},
			{
				ID:        "gpt-4.1",
				Object:    "model",
				Created:   1677657600,
				OwnedBy:   "openai",
				SecretKey: os.Getenv("LLM_API_KEY"),
			},
		},
	}

	for _, model := range MODELS.Data {
		ModelMap[model.ID] = model
	}
}

func NewProxyConfigFromEnv(providerOverride string) *ProxyConfig {
	provider := providerOverride
	if provider == "" {
		provider = os.Getenv("LLM_PROVIDER")
	}
	switch provider {
	case "aws":
		return &ProxyConfig{
			BaseURL: "https://api.aws.com",
			Name:    "AWS",
		}
	case "ollama":
		return &ProxyConfig{
			BaseURL: "http://localhost:11434",
			Name:    "Ollama",
		}
	default: // openai
		return &ProxyConfig{
			BaseURL: "https://api.openai.com",
			Name:    "OpenAI",
		}
	}
}
