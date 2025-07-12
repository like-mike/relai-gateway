package routes

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/like-mike/relai-gateway/metrics"
)

func PrometheusMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		route := c.OriginalURL()
		err := c.Next()
		latency := time.Since(start).Seconds()
		code := c.Response().StatusCode()
		status := fmt.Sprintf("%d", code)

		fmt.Printf("PrometheusMiddleware: route=%s, status=%s, latency=%.4f\n", route, status, latency)

		metrics.HttpRequestsTotal.WithLabelValues(status, route).Inc()
		metrics.HttpRequestDurationSeconds.WithLabelValues(route).Observe(latency)
		metrics.HttpResponseCodesTotal.WithLabelValues(status, route).Inc()
		if err != nil {
			metrics.HttpErrorsTotal.WithLabelValues(err.Error(), route).Inc()
			fmt.Printf("PrometheusMiddleware: error=%v\n", err)
		}
		return err
	}
}
