package usage

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/like-mike/relai-gateway/shared/models"
	"github.com/pkoukk/tiktoken-go"
)

// TiktokenExtractor uses OpenAI's official tiktoken for accurate token counting
type TiktokenExtractor struct {
	modelID string
}

// NewTiktokenExtractor creates a new tiktoken-based extractor
func NewTiktokenExtractor(modelID string) *TiktokenExtractor {
	return &TiktokenExtractor{
		modelID: modelID,
	}
}

// ExtractFromStreamingResponse counts tokens accurately for streaming responses
func (e *TiktokenExtractor) ExtractFromStreamingResponse(responseBody []byte, requestBody []byte) (*models.AIProviderUsage, error) {
	// Extract prompt from request
	promptText, err := e.extractPromptFromRequest(requestBody)
	if err != nil {
		return nil, err
	}

	// Extract completion from streaming response
	completionText, err := e.extractCompletionFromStream(responseBody)
	if err != nil {
		return nil, err
	}

	// Count tokens accurately with tiktoken
	promptTokens, err := e.countTokens(promptText)
	if err != nil {
		log.Printf("Failed to count prompt tokens, using estimation: %v", err)
		promptTokens = e.estimateTokens(promptText)
	}

	completionTokens, err := e.countTokens(completionText)
	if err != nil {
		log.Printf("Failed to count completion tokens, using estimation: %v", err)
		completionTokens = e.estimateTokens(completionText)
	}

	log.Printf("Tiktoken usage - Prompt: %d tokens, Completion: %d tokens (model: %s)",
		promptTokens, completionTokens, e.modelID)

	return &models.AIProviderUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}, nil
}

// countTokens uses tiktoken for accurate token counting
func (e *TiktokenExtractor) countTokens(text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// Get the appropriate encoding for the model
	encodingName := e.getEncodingForModel()

	tkm, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return 0, err
	}

	tokens := tkm.Encode(text, nil, nil)
	return len(tokens), nil
}

// getEncodingForModel returns the correct tiktoken encoding for the model
func (e *TiktokenExtractor) getEncodingForModel() string {
	modelID := strings.ToLower(e.modelID)

	switch {
	case strings.Contains(modelID, "gpt-4"):
		return "cl100k_base"
	case strings.Contains(modelID, "gpt-3.5-turbo"):
		return "cl100k_base"
	case strings.Contains(modelID, "text-davinci-003"):
		return "p50k_base"
	case strings.Contains(modelID, "text-davinci-002"):
		return "p50k_base"
	case strings.Contains(modelID, "code"):
		return "p50k_base"
	default:
		// Default to GPT-4 encoding for unknown models
		return "cl100k_base"
	}
}

// extractPromptFromRequest extracts prompt text from request body
func (e *TiktokenExtractor) extractPromptFromRequest(requestBody []byte) (string, error) {
	var request map[string]interface{}
	if err := json.Unmarshal(requestBody, &request); err != nil {
		return "", err
	}

	// Handle chat completion format (most common)
	if messages, ok := request["messages"].([]interface{}); ok {
		var fullPrompt strings.Builder
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok {
					if content, ok := msgMap["content"].(string); ok {
						// Format like OpenAI does internally
						fullPrompt.WriteString(role)
						fullPrompt.WriteString(": ")
						fullPrompt.WriteString(content)
						fullPrompt.WriteString("\n")
					}
				}
			}
		}
		return fullPrompt.String(), nil
	}

	// Handle legacy completion format
	if prompt, ok := request["prompt"].(string); ok {
		return prompt, nil
	}

	return "", errors.New("could not extract prompt from request")
}

// extractCompletionFromStream extracts completion text from streaming response
func (e *TiktokenExtractor) extractCompletionFromStream(responseBody []byte) (string, error) {
	lines := strings.Split(string(responseBody), "\n")
	var completion strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")

			if jsonStr == "[DONE]" {
				continue
			}

			if !json.Valid([]byte(jsonStr)) {
				continue
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
				continue
			}

			// Extract content from streaming chunk
			if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					// Chat completion delta
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							completion.WriteString(content)
						}
					}
					// Legacy completion text
					if text, ok := choice["text"].(string); ok {
						completion.WriteString(text)
					}
				}
			}
		}
	}

	return completion.String(), nil
}

// estimateTokens provides fallback estimation if tiktoken fails
func (e *TiktokenExtractor) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Simple estimation: ~4 characters per token
	// This is roughly accurate for English text
	return len(text) / 4
}
