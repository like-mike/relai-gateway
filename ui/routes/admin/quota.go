package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetQuotaHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "quota_cards.html", gin.H{
		"TotalUsage":     1200020,
		"RemainingQuota": 380000,
		"PercentUsed":    "245%",
		"userEmail":      "blah",
	})
}
