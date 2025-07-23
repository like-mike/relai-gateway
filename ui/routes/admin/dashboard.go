package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/ui/auth"
)

func DashboardHandler(c *gin.Context) {
	userData := auth.GetUserContext(c)
	userData["activePage"] = "api_keys"
	userData["title"] = "API Keys"

	c.HTML(http.StatusOK, "api-keys.html", userData)
}
