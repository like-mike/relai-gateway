package proxy

import (
	"net/http"

	"github.com/like-mike/relai-gateway/provider"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func recordTracingMetadata(cfg *provider.ProxyConfig, span, childSpan trace.Span, req *http.Request, body []byte) {
	authHeader := req.Header.Get("Authorization")

	span.SetAttributes(
		attribute.String("llm.provider", cfg.Name),
		attribute.String("llm.auth_header", authHeader),
	)

	childSpan.SetAttributes(
		attribute.String("llm.provider", cfg.Name),
		attribute.String("llm.endpoint", req.URL.String()),
		attribute.String("llm.auth_header", authHeader),
		attribute.Int("llm.request.size_bytes", len(body)),
	)

	if cfg.Name == "openai" {
		childSpan.SetAttributes(
			attribute.String("llm.request.body", string(body)),
			attribute.Int("llm.request.body.size_bytes", len(body)),
		)
	}
}
