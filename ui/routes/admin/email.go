rpackage admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/ui/auth"
)

// EmailPageHandler handles the email management page
func EmailPageHandler(c *gin.Context) {
	userData := auth.GetUserContext(c)
	userData["activePage"] = "email"
	userData["title"] = "Email Management"

	c.HTML(http.StatusOK, "email.html", userData)
}
