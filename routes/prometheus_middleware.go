package routes

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"status", "route"})
	httpRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})
	httpErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"error", "route"})
	httpResponseCodesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_response_codes_total",
		Help: "Total number of HTTP response codes",
	}, []string{"code", "route"})
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

		httpRequestsTotal.WithLabelValues(status, route).Inc()
		httpRequestDurationSeconds.WithLabelValues(route).Observe(latency)
		httpResponseCodesTotal.WithLabelValues(status, route).Inc()
		if err != nil {
			httpErrorsTotal.WithLabelValues(err.Error(), route).Inc()
			fmt.Printf("PrometheusMiddleware: error=%v\n", err)
		}
		return err
	}
}
