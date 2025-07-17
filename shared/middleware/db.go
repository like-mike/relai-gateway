package middleware

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

const DBKey = "db"

func DBMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(DBKey, db)
		c.Next()
	}
}

// Helper to get DB from Gin context
func GetDB(c *gin.Context) *sql.DB {
	db, ok := c.MustGet(DBKey).(*sql.DB)
	if !ok {
		panic("DB not found in Gin context")
	}
	return db
}
