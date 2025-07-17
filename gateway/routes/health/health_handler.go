package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler returns a simple health check.
func Handler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}
