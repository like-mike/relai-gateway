package admin

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/shared/middleware"
)

type APIKeyRow struct {
	ID        string
	Key       string
	OrgName   string
	UserEmail string
	CreatedAt string
	ExpiresAt sql.NullString
	Active    bool
	MaxTokens int
}

func APIKeysHandler(c *gin.Context) {
	db := middleware.GetDB(c)

	rows, err := db.Query(`
		SELECT
			k.id,
			k.key,
			o.name AS org_name,
			u.email AS user_email,
			k.created_at,
			k.expires_at,
			k.active,
			k.max_tokens
		FROM api_keys k
		JOIN orgs o ON o.id = k.org_id
		JOIN users u ON u.id = k.user_id
	`)
	if err != nil {
		log.Println("Query error:", err)
		c.String(http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	var keys []APIKeyRow
	for rows.Next() {
		var k APIKeyRow
		err := rows.Scan(&k.ID, &k.Key, &k.OrgName, &k.UserEmail, &k.CreatedAt, &k.ExpiresAt, &k.Active, &k.MaxTokens)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		keys = append(keys, k)
	}

	// Render only the table rows for htmx
	// Extract user email from session cookie
	userEmail := ""
	if session, err := c.Cookie("session"); err == nil && session != "" {
		userEmail = session
	}

	if userEmail == "" {
		log.Println("No user email found in session cookie")
	}

	fmt.Println(userEmail)
	c.HTML(http.StatusOK, "api_keys_table.html", gin.H{
		"APIKeys": keys,
	})
}
