package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler exposes Prometheus metrics.
func MetricsHandler(c *gin.Context) {
	c.Status(http.StatusOK)
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
