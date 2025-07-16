// routes/proxy.go
package proxy

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var sharedClient = &http.Client{
	Timeout: 60 * time.Second, // optional
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	},
}

func Handler(c *gin.Context) {
	ctx := c.Request.Context()
	tracer := otel.GetTracerProvider().Tracer("gateway")

	// method := c.Request.Method
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery
	target := path
	if query != "" {
		target += "?" + query
	}

	cfg := provider.NewProxyConfigFromEnv("openai")

	// Build proxy request
	req, bodyBytes, err := prepareRequest(cfg, c, target)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Trace the provider call
	ctx, spanInvoke := tracer.Start(ctx, "invoke_provider")
	defer spanInvoke.End()

	ctx, spanExec := tracer.Start(ctx, cfg.Name)
	defer spanExec.End()

	recordTracingMetadata(cfg, spanInvoke, spanExec, req, bodyBytes)

	// Send request
	start := time.Now()
	// client := &http.Client{}
	resp, err := sharedClient.Do(req)
	duration := time.Since(start).Milliseconds()
	spanInvoke.SetAttributes(attribute.Int64("llm.request.duration_ms", duration))

	// Build response
	writeDownstreamResponse(c, resp, err, tracer)
}
