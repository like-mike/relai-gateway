# RelAI Gateway API Documentation

This directory contains comprehensive documentation and testing tools for the RelAI Gateway API.

## Files Overview

- **`swagger.yaml`** - Complete OpenAPI 3.0 specification
- **`api-tests.http`** - HTTP request collection for testing
- **`README.md`** - This documentation file

## Quick Start

### 1. Get an API Key

1. Access the admin UI at `http://localhost:8080/admin`
2. Navigate to the API Keys section
3. Create a new API key for your organization
4. Copy the generated API key (starts with `sk-`)

### 2. Test the API

#### Using the HTTP Collection
1. Open `api-tests.http` in VSCode with the REST Client extension
2. Replace `@apiKey` variable at the top with your actual API key
3. Click "Send Request" on any endpoint you want to test

#### Using curl
```bash
# Test health endpoint (no auth required)
curl http://localhost:8081/health

# Test models endpoint with authentication
curl -H "Authorization: Bearer YOUR_API_KEY" \
     http://localhost:8081/v1/models

# Test chat completion
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8081/v1/chat/completions \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "Hello!"}],
       "max_tokens": 50
     }'
```

## API Endpoints

### Public Endpoints (No Authentication)
- `GET /health` - Health check
- `GET /v1/models` - List all available models (public discovery)
- `GET /models` - Alternative models endpoint

### Authenticated Endpoints (Require API Key)
- `GET /v1/models` - List organization's accessible models (when authenticated)
- `POST /v1/chat/completions` - Chat completions
- `POST /v1/completions` - Text completions
- `POST /v1/embeddings` - Text embeddings
- `POST /v1/moderations` - Content moderation
- `POST /v1/images/generations` - Image generation
- `POST /v1/audio/transcriptions` - Audio transcription
- `POST /v1/audio/translations` - Audio translation

### Custom Endpoints
- `POST /api/{custom_endpoint}` - Organization-specific custom endpoints

## Authentication

All API endpoints (except health and public model listing) require authentication via API key:

```
Authorization: Bearer sk-your-api-key-here
```

### Organization-Based Access Control

Each API key belongs to an organization and only provides access to:
- Models that the organization has been granted access to
- Custom endpoints configured for that organization

## Testing Different API Pass-throughs

### 1. Standard OpenAI API Compatibility

Test that the gateway correctly proxies standard OpenAI API calls:

```bash
# Test chat completions
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8081/v1/chat/completions \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "Test message"}]
     }'

# Test text completions
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8081/v1/completions \
     -d '{
       "model": "gpt-3.5-turbo",
       "prompt": "Complete this sentence:",
       "max_tokens": 50
     }'
```

### 2. Custom Organization Endpoints

Test organization-specific endpoints:

```bash
# Create a custom endpoint in the UI first, then test:
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8081/api/your-custom-endpoint \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "Custom endpoint test"}]
     }'
```

### 3. Model Access Control Testing

Test that model access is properly restricted by organization:

```bash
# This should only return models your organization has access to
curl -H "Authorization: Bearer YOUR_API_KEY" \
     http://localhost:8081/v1/models

# Compare with unauthenticated request (shows all models)
curl http://localhost:8081/v1/models
```

### 4. Error Handling Testing

```bash
# Test invalid API key
curl -H "Authorization: Bearer sk-invalid" \
     http://localhost:8081/v1/models

# Test missing API key on protected endpoint
curl -X POST http://localhost:8081/v1/chat/completions \
     -d '{"model": "gpt-3.5-turbo", "messages": []}'

# Test nonexistent custom endpoint
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -X POST http://localhost:8081/api/nonexistent
```

## Swagger UI

To view the interactive API documentation:

1. **Online Swagger Editor**:
   - Go to https://editor.swagger.io/
   - Copy the contents of `swagger.yaml`
   - Paste into the editor

2. **Local Swagger UI** (if you have it installed):
   ```bash
   swagger-ui-serve docs/swagger.yaml
   ```

3. **Docker Swagger UI**:
   ```bash
   docker run -p 8082:8080 -v $(pwd)/docs:/docs \
     -e SWAGGER_JSON=/docs/swagger.yaml swaggerapi/swagger-ui
   ```
   Then open http://localhost:8082

## Custom Endpoint Configuration

### Creating Custom Endpoints

1. Access the admin UI at `http://localhost:8080/admin/models`
2. Click on the "Custom Endpoints" tab
3. Click "Add Endpoint"
4. Configure:
   - **Name**: Display name for the endpoint
   - **Path Prefix**: URL path (e.g., "chat" creates `/api/chat`)
   - **Organization**: Select the organization
   - **Primary Model**: Main model to use
   - **Fallback Model**: Backup model if primary fails
   - **Description**: Optional description

### Using Custom Endpoints

Once created, custom endpoints work like standard API endpoints but with organization-specific routing:

```bash
# If you created an endpoint with path prefix "support-bot"
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8081/api/support-bot \
     -d '{
       "model": "gpt-3.5-turbo",
       "messages": [{"role": "user", "content": "I need help"}]
     }'
```

## Error Responses

The API returns standard HTTP status codes and JSON error responses:

```json
{
  "error": "Invalid API key",
  "code": "authentication_failed"
}
```

Common error codes:
- `401` - Authentication required or invalid API key
- `403` - Access denied (organization doesn't have model access)
- `404` - Custom endpoint not found
- `400` - Bad request (malformed JSON, missing parameters)
- `500` - Internal server error

## Rate Limiting and Quotas

Rate limiting and quota management can be configured per organization through the admin UI. Check your organization's quota status in the admin dashboard.

## Development and Testing

### Adding New API Pass-throughs

1. Add the new route to `gateway/app.go`
2. Update the proxy handler if needed
3. Add the endpoint to `swagger.yaml`
4. Add test cases to `api-tests.http`
5. Update this documentation

### Testing with Different Providers

The gateway supports multiple AI providers. Test with different providers by:
1. Configuring models with different providers in the admin UI
2. Setting the appropriate environment variables for each provider
3. Testing the same API calls with different models

## Monitoring and Logging

The gateway includes comprehensive logging and monitoring:
- Request/response logging
- Authentication events
- Model access tracking
- Error tracking
- Performance metrics

Check the gateway logs for detailed information about API usage and any issues.