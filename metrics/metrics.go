package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"status", "route"})
	HttpRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})
	HttpErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"error", "route"})
	HttpResponseCodesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_response_codes_total",
		Help: "Total number of HTTP response codes",
	}, []string{"code", "route"})
	LlmTokens = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llm_tokens",
		Help:    "Number of LLM tokens per completion",
		Buckets: prometheus.LinearBuckets(0, 50, 20),
	}, []string{"route"})
)
