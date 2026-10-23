package middleware

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ContentTypeChecker is a middleware to check content type.
func ContentTypeChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ParseMediaType drops parameters, so "application/json; charset=utf-8"
		// (what browsers and axios often send) is accepted too.
		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Specified content type not allowed."})
			c.Abort()
			return
		}
		c.Next()
	}
}
