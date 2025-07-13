package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler returns a simple health check.
func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}
