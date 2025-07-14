package middleware

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// TracingMiddleware adds request body to OpenTelemetry span.
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/metrics" {
			c.Next()
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		// fmt.Println(string(body))
		ctx, span := otel.GetTracerProvider().Tracer("gin").Start(c.Request.Context(), "request")
		span.SetAttributes(attribute.String("http.request.body", string(body)))
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
