package models

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/gateway/provider"
)

func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, provider.MODELS)
}
