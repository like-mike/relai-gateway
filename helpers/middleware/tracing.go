package middleware

import (
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
		// body, _ := io.ReadAll(c.Request.Body)
		// c.Request.Body = io.NopCloser(bytes.NewReader(body))
		// fmt.Println(string(body))
		ctx, span := otel.GetTracerProvider().Tracer("gateway").Start(c.Request.Context(), "handle_request")
		span.SetAttributes(attribute.String("http.request.body", "blah"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		span.End()
	}
}
