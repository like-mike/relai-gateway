package provider

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
)

// ChatMessage represents a single chat message with role and content.
type ChatMessage struct {
	Role    string
	Content string
}

// CompletionRequest represents the input for a completion request.
// For chat models, use Messages.
type CompletionRequest struct {
	Prompt    string
	MaxTokens int
	Messages  []ChatMessage
}

// CompletionResponse represents the text output from a completion request.
type CompletionResponse struct {
	Text string
}

// CompletionProvider defines an interface for completion services.
type CompletionProvider interface {
	GetCompletions(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

// OpenAIProvider is an implementation of CompletionProvider using OpenAI's API.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider constructs an OpenAIProvider using environment vars:
func NewOpenAIProvider(apiKey, llmModel string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}
	if llmModel == "" {
		llmModel = "gpt-3.5-turbo"
	}
	client := openai.NewClient(apiKey)
	return &OpenAIProvider{
		client: client,
		model:  llmModel,
	}, nil
}

// GetCompletions selects chat or text completions based on request fields.
func (p *OpenAIProvider) GetCompletions(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if len(req.Messages) > 0 {
		messages := make([]openai.ChatCompletionMessage, len(req.Messages))
		for i, m := range req.Messages {
			messages[i] = openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
		}
		chatReq := openai.ChatCompletionRequest{Model: p.model, Messages: messages}
		chatResp, err := p.client.CreateChatCompletion(ctx, chatReq)
		if err != nil {
			return nil, err
		}
		if len(chatResp.Choices) == 0 {
			return nil, errors.New("no chat choices returned")
		}
		return &CompletionResponse{Text: chatResp.Choices[0].Message.Content}, nil
	}
	textReq := openai.CompletionRequest{Model: p.model, Prompt: []string{req.Prompt}, MaxTokens: req.MaxTokens}
	textResp, err := p.client.CreateCompletion(ctx, textReq)
	if err != nil {
		return nil, err
	}
	if len(textResp.Choices) == 0 {
		return nil, errors.New("no choices returned")
	}
	return &CompletionResponse{Text: textResp.Choices[0].Text}, nil
}
