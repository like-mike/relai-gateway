#!/bin/bash

# RelAI Gateway API Test Script
# This script tests the various API endpoints and pass-throughs

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8081}"
API_KEY="${API_KEY:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Test function
test_endpoint() {
    local method="$1"
    local url="$2"
    local headers="$3"
    local data="$4"
    local description="$5"
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "→ $method $url"
    
    if [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" $headers -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" $headers)
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        print_success "Status: $http_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    elif [[ "$http_code" -ge 400 && "$http_code" -lt 500 ]]; then
        print_warning "Status: $http_code (Client Error - Expected for some tests)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        print_error "Status: $http_code"
        echo "$body"
    fi
}

# Check if jq is available
if ! command -v jq &> /dev/null; then
    print_warning "jq is not installed. JSON responses will not be formatted."
fi

print_header "RelAI Gateway API Test Suite"
echo "Gateway URL: $GATEWAY_URL"

if [[ -z "$API_KEY" ]]; then
    print_warning "No API_KEY environment variable set. Some tests will fail."
    echo "Set your API key with: export API_KEY=sk-your-api-key-here"
else
    echo "API Key: ${API_KEY:0:10}..."
fi

# Test 1: Health Check (No Auth)
print_header "System Health Tests"
test_endpoint "GET" "$GATEWAY_URL/health" "" "" "Health check endpoint"

# Test 2: Models Endpoint (No Auth)
print_header "Public Model Listing Tests"
test_endpoint "GET" "$GATEWAY_URL/v1/models" "" "" "Models endpoint (no auth)"
test_endpoint "GET" "$GATEWAY_URL/models" "" "" "Alternative models endpoint"

if [[ -n "$API_KEY" ]]; then
    # Test 3: Models Endpoint (With Auth)
    print_header "Authenticated Model Listing Tests"
    test_endpoint "GET" "$GATEWAY_URL/v1/models" "-H 'Authorization: Bearer $API_KEY'" "" "Models endpoint (with auth)"
    
    # Test 4: Chat Completions
    print_header "Chat Completion Tests"
    chat_data='{
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "Hello! This is a test message."}
        ],
        "max_tokens": 50,
        "temperature": 0.7
    }'
    test_endpoint "POST" "$GATEWAY_URL/v1/chat/completions" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$chat_data" "Chat completion"
    
    # Test 5: Text Completions
    print_header "Text Completion Tests"
    completion_data='{
        "model": "gpt-3.5-turbo",
        "prompt": "The future of AI is",
        "max_tokens": 30,
        "temperature": 0.5
    }'
    test_endpoint "POST" "$GATEWAY_URL/v1/completions" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$completion_data" "Text completion"
    
    # Test 6: Embeddings
    print_header "Embeddings Tests"
    embedding_data='{
        "model": "text-embedding-ada-002",
        "input": "This is a test sentence for embedding."
    }'
    test_endpoint "POST" "$GATEWAY_URL/v1/embeddings" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$embedding_data" "Text embeddings"
    
    # Test 7: Moderations
    print_header "Moderation Tests"
    moderation_data='{
        "input": "This is a safe test message."
    }'
    test_endpoint "POST" "$GATEWAY_URL/v1/moderations" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$moderation_data" "Content moderation"
    
    # Test 8: Custom Endpoints (These will likely fail unless configured)
    print_header "Custom Endpoint Tests"
    custom_data='{
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "Test custom endpoint"}
        ],
        "max_tokens": 30
    }'
    test_endpoint "POST" "$GATEWAY_URL/api/chat" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$custom_data" "Custom chat endpoint"
    test_endpoint "POST" "$GATEWAY_URL/api/assistant" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$custom_data" "Custom assistant endpoint"
fi

# Test 9: Error Cases
print_header "Error Handling Tests"
test_endpoint "GET" "$GATEWAY_URL/v1/models" "-H 'Authorization: Bearer sk-invalid-key'" "" "Invalid API key test"
test_endpoint "POST" "$GATEWAY_URL/v1/chat/completions" "-H 'Content-Type: application/json'" '{"model": "gpt-3.5-turbo", "messages": []}' "Missing API key test"
test_endpoint "POST" "$GATEWAY_URL/api/nonexistent-endpoint" "-H 'Authorization: Bearer $API_KEY' -H 'Content-Type: application/json'" "$chat_data" "Nonexistent custom endpoint test"

# Test 10: Load Test (Multiple quick requests)
if [[ -n "$API_KEY" ]]; then
    print_header "Basic Load Tests"
    echo "Sending 5 quick requests..."
    
    quick_data='{
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Quick test"}],
        "max_tokens": 10
    }'
    
    for i in {1..5}; do
        echo -n "Request $i: "
        response=$(curl -s -w "%{http_code}" -X POST "$GATEWAY_URL/v1/chat/completions" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$quick_data")
        
        http_code=$(echo "$response" | tail -c 4)
        if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
            print_success "$http_code"
        else
            print_error "$http_code"
        fi
    done
fi

print_header "Test Summary"
echo "API testing complete!"
echo ""
echo "To run specific tests:"
echo "  Health only:     curl $GATEWAY_URL/health"
echo "  Models only:     curl $GATEWAY_URL/v1/models"
echo "  With your key:   export API_KEY=sk-your-key && $0"
echo ""
echo "For interactive testing, use the HTTP collection in docs/api-tests.http"
echo "For full documentation, see docs/README.md"