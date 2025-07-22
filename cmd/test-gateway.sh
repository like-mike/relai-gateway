#!/bin/bash

# Test script for RelAI Gateway with database integration
echo "=== Testing RelAI Gateway with Database Integration ==="

GATEWAY_URL="http://localhost:8081"
UI_URL="http://localhost:8080"

echo "Testing Gateway Health Check..."
curl -s "$GATEWAY_URL/health" | jq '.' || echo "Health check failed"

echo -e "\n\nTesting Models Endpoint (no auth required)..."
curl -s "$GATEWAY_URL/v1/models" | jq '.' || echo "Models endpoint failed"

echo -e "\n\nTesting Models Endpoint Alternative (no auth required)..."
curl -s "$GATEWAY_URL/models" | jq '.' || echo "Models endpoint failed"

echo -e "\n\n=== API KEY REQUIRED FOR ALL OTHER ENDPOINTS ==="
echo "All proxy endpoints now require valid API keys from the database!"

echo -e "\n\nTo test with API key authentication:"
echo "1. First create an API key in the UI at: $UI_URL/admin"
echo "2. Copy the generated API key (starts with sk-...)"

echo -e "\n\nTesting Standard OpenAI API endpoints (requires API key):"
echo "curl -H 'Authorization: Bearer YOUR_API_KEY' $GATEWAY_URL/v1/chat/completions"
echo "curl -H 'Authorization: Bearer YOUR_API_KEY' $GATEWAY_URL/v1/completions"
echo "curl -H 'Authorization: Bearer YOUR_API_KEY' $GATEWAY_URL/v1/embeddings"

echo -e "\n\nTesting Custom Endpoints (requires API key):"
echo "1. Create a custom endpoint in the UI at: $UI_URL/admin/models (Endpoints tab)"
echo "2. Test with: curl -H 'Authorization: Bearer YOUR_API_KEY' $GATEWAY_URL/api/YOUR_CUSTOM_PREFIX"

echo -e "\n\nExample Chat Completion Request (replace YOUR_API_KEY with actual key):"
echo 'curl -H "Authorization: Bearer YOUR_API_KEY" \\'
echo "     -H \"Content-Type: application/json\" \\"
echo "     -X POST $GATEWAY_URL/v1/chat/completions \\"
echo "     -d '{\"model\":\"gpt-3.5-turbo\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'"

echo -e "\n\nExample Custom Endpoint Request (replace YOUR_API_KEY and custom-endpoint):"
echo 'curl -H "Authorization: Bearer YOUR_API_KEY" \\'
echo "     -H \"Content-Type: application/json\" \\"
echo "     -X POST $GATEWAY_URL/api/custom-endpoint \\"
echo "     -d '{\"model\":\"gpt-3.5-turbo\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'"

echo -e "\n\nNote: All API keys are validated against the database and must belong to an active organization."