# relai-gateway

A simple, extensible Go API gateway for LLM completions.

## Overview

This service provides a `/completions` endpoint that delegates to a pluggable LLM provider (default: OpenAI).

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/like-mike/relai-gateway.git
   cd relai-gateway
   ```
2. Set required environment variables:
   ```bash
   export OPENAI_API_KEY=your_api_key
   export OPENAI_MODEL=text-davinci-003      # Optional, defaults to text-davinci-003
   export LLM_PROVIDER=openai                # Optional, default provider
   ```
+  Alternatively, create a `.env` file in the project root with these variables; it will be loaded automatically and is included in .gitignore.

## Run

Start the server on port 8080:
```bash
go run main.go
```

POST JSON to `http://localhost:8080/completions`:
```json
{
  "prompt": "Hello, world!",
  "max_tokens": 50
}
```

## Extensibility

- Define new LLM providers by implementing the `CompletionProvider` interface.
- Add provider initialization logic in `provider_factory.go`.

## Testing

Run unit tests:
```bash
go test ./...
