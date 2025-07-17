package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler exposes Prometheus metrics.
func Handler(c *gin.Context) {
	c.Status(http.StatusOK)
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
