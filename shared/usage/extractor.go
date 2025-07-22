package usage

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/like-mike/relai-gateway/shared/models"
)

// UsageExtractor extracts usage information from provider responses
type UsageExtractor interface {
	ExtractUsage(responseBody []byte) (*models.AIProviderUsage, error)
	GetProviderName() string
}

// OpenAIExtractor extracts usage from OpenAI API responses
type OpenAIExtractor struct{}

func (e *OpenAIExtractor) GetProviderName() string {
	return "openai"
}

func (e *OpenAIExtractor) ExtractUsage(responseBody []byte) (*models.AIProviderUsage, error) {
	// Skip empty or very small responses
	if len(responseBody) < 10 {
		return nil, errors.New("response too small to contain usage data")
	}

	// Try to decompress if response appears to be gzipped
	decompressedBody, err := e.decompressIfNeeded(responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress response: %w", err)
	}

	// Use decompressed body for further processing
	responseBody = decompressedBody

	// Log response info for debugging (only if extraction fails)
	defer func() {
		if r := recover(); r != nil {
			preview := string(responseBody)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			log.Printf("OpenAI extractor panic - response preview: %s", preview)
			panic(r)
		}
	}()

	// Check for streaming response patterns - no longer try to parse, just fail fast
	if strings.Contains(string(responseBody), "data: ") || strings.HasPrefix(string(responseBody), "data:") {
		return nil, errors.New("streaming response detected - use tiktoken extractor instead")
	}

	// Check if response looks like JSON
	if !json.Valid(responseBody) {
		// Log sample of invalid response for debugging
		preview := string(responseBody)
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		log.Printf("Invalid JSON response sample: %s", preview)
		return nil, fmt.Errorf("response is not valid JSON (length: %d)", len(responseBody))
	}

	var response struct {
		Usage models.AIProviderUsage `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate that we got meaningful usage data
	if response.Usage.TotalTokens == 0 && response.Usage.PromptTokens == 0 && response.Usage.CompletionTokens == 0 {
		return nil, errors.New("no usage data found in OpenAI response")
	}

	return &response.Usage, nil
}

// decompressIfNeeded checks for various compression formats and decompresses them
func (e *OpenAIExtractor) decompressIfNeeded(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return data, nil
	}

	// Log hex dump of first 32 bytes for debugging
	debugHex := hex.EncodeToString(data[:min(32, len(data))])
	log.Printf("Response hex dump (first 32 bytes): %s", debugHex)

	// Try gzip first (magic: 1f 8b)
	if data[0] == 0x1f && data[1] == 0x8b {
		log.Printf("Detected gzip compression, decompressing...")
		return e.decompressGzip(data)
	}

	// Try deflate/zlib (magic: 78 xx where xx can be 01, 5e, 9c, da)
	if data[0] == 0x78 && (data[1] == 0x01 || data[1] == 0x5e || data[1] == 0x9c || data[1] == 0xda) {
		log.Printf("Detected zlib/deflate compression, decompressing...")
		return e.decompressZlib(data)
	}

	// Try Brotli decompression (binary format, no clear magic number)
	if !e.isPrintableText(data[:min(100, len(data))]) {
		log.Printf("Attempting Brotli decompression...")
		if decompressed, err := e.decompressBrotli(data); err == nil {
			return decompressed, nil
		}

		log.Printf("Attempting raw deflate decompression...")
		if decompressed, err := e.decompressRawDeflate(data); err == nil {
			return decompressed, nil
		}

		// If all decompression attempts fail, log detailed info
		log.Printf("Failed all decompression attempts. Response details:")
		log.Printf("  Length: %d bytes", len(data))
		log.Printf("  First 4 bytes: %02x %02x %02x %02x", data[0], data[1], data[2], data[3])
		log.Printf("  Appears to be binary/compressed data")

		return nil, errors.New("response appears to be compressed in unsupported format")
	}

	// Return original data if it appears to be text
	return data, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decompressGzip decompresses gzip data
func (e *OpenAIExtractor) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
	}

	log.Printf("Successfully decompressed gzip: %d bytes -> %d bytes", len(data), len(decompressed))
	return decompressed, nil
}

// decompressZlib decompresses zlib/deflate data with headers
func (e *OpenAIExtractor) decompressZlib(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress zlib data: %w", err)
	}

	log.Printf("Successfully decompressed zlib: %d bytes -> %d bytes", len(data), len(decompressed))
	return decompressed, nil
}

// decompressRawDeflate attempts raw deflate decompression (no headers)
func (e *OpenAIExtractor) decompressRawDeflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress raw deflate data: %w", err)
	}

	decompressed := buf.Bytes()
	log.Printf("Successfully decompressed raw deflate: %d bytes -> %d bytes", len(data), len(decompressed))
	return decompressed, nil
}

// decompressBrotli attempts Brotli decompression
func (e *OpenAIExtractor) decompressBrotli(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress Brotli data: %w", err)
	}

	log.Printf("Successfully decompressed Brotli: %d bytes -> %d bytes", len(data), len(decompressed))
	return decompressed, nil
}

// isPrintableText checks if the data contains mostly printable ASCII characters
func (e *OpenAIExtractor) isPrintableText(data []byte) bool {
	printableCount := 0
	for _, b := range data {
		if b >= 32 && b <= 126 || b == 9 || b == 10 || b == 13 { // printable ASCII + tab/newline/CR
			printableCount++
		}
	}
	return float64(printableCount)/float64(len(data)) > 0.7 // 70% printable characters
}

// extractFromStreamingResponse has been removed - use tiktoken_extractor.go instead
// This function is no longer used since we simplified the streaming approach
// Streaming responses now use tiktoken for direct token counting

// AnthropicExtractor extracts usage from Anthropic API responses
type AnthropicExtractor struct{}

func (e *AnthropicExtractor) GetProviderName() string {
	return "anthropic"
}

func (e *AnthropicExtractor) ExtractUsage(responseBody []byte) (*models.AIProviderUsage, error) {
	var response struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	// Map Anthropic format to our standard format
	usage := &models.AIProviderUsage{
		PromptTokens:     response.Usage.InputTokens,
		CompletionTokens: response.Usage.OutputTokens,
		TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
	}

	// Validate that we got meaningful usage data
	if usage.TotalTokens == 0 {
		return nil, errors.New("no usage data found in Anthropic response")
	}

	return usage, nil
}

// GenericExtractor fallback extractor for unknown providers
type GenericExtractor struct {
	providerName string
}

func NewGenericExtractor(providerName string) *GenericExtractor {
	return &GenericExtractor{providerName: providerName}
}

func (e *GenericExtractor) GetProviderName() string {
	return e.providerName
}

func (e *GenericExtractor) ExtractUsage(responseBody []byte) (*models.AIProviderUsage, error) {
	// Try common patterns for usage extraction
	patterns := []string{"usage", "token_usage", "tokens"}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseMap); err != nil {
		return nil, err
	}

	for _, pattern := range patterns {
		if usageData, exists := responseMap[pattern]; exists {
			if usageMap, ok := usageData.(map[string]interface{}); ok {
				usage := &models.AIProviderUsage{}

				if promptTokens, ok := usageMap["prompt_tokens"].(float64); ok {
					usage.PromptTokens = int(promptTokens)
				}
				if completionTokens, ok := usageMap["completion_tokens"].(float64); ok {
					usage.CompletionTokens = int(completionTokens)
				}
				if totalTokens, ok := usageMap["total_tokens"].(float64); ok {
					usage.TotalTokens = int(totalTokens)
				}

				// Calculate total if not provided
				if usage.TotalTokens == 0 && (usage.PromptTokens > 0 || usage.CompletionTokens > 0) {
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				}

				if usage.TotalTokens > 0 {
					return usage, nil
				}
			}
		}
	}

	return nil, errors.New("no usage data found in provider response")
}

// ExtractorFactory creates the appropriate extractor for a provider
type ExtractorFactory struct{}

func NewExtractorFactory() *ExtractorFactory {
	return &ExtractorFactory{}
}

func (f *ExtractorFactory) GetExtractor(provider string) UsageExtractor {
	switch provider {
	case "openai":
		return &OpenAIExtractor{}
	case "anthropic":
		return &AnthropicExtractor{}
	default:
		log.Printf("Unknown provider '%s', using generic extractor", provider)
		return NewGenericExtractor(provider)
	}
}

// ExtractUsageFromResponse extracts usage data from a provider response
func ExtractUsageFromResponse(responseBody []byte, provider string) (*models.AIProviderUsage, error) {
	factory := NewExtractorFactory()
	extractor := factory.GetExtractor(provider)
	return extractor.ExtractUsage(responseBody)
}
