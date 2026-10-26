package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	// authUserIDKey is the gin context key under which the logged-in user's ID is stored.
	authUserIDKey = "auth_user_id"
)

// AuthMiddleware rejects the request with 401 unless it carries a valid
// "Authorization: Bearer <token>" header. On success it stores the user ID
// from the token in the context; read it with AuthUserID.
func AuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(authorizationHeader)
		if !strings.HasPrefix(header, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or malformed Authorization header. Expected: Bearer <token>."})
			return
		}
		if !setUserFromToken(c, tokenMaker, header) {
			return
		}
		c.Next()
	}
}

// OptionalAuthMiddleware lets anonymous requests through, but when an
// Authorization header is present it must be valid (401 otherwise, so the
// client knows to refresh its token).
func OptionalAuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(authorizationHeader)
		if header == "" {
			c.Next()
			return
		}
		if !strings.HasPrefix(header, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed Authorization header. Expected: Bearer <token>."})
			return
		}
		if !setUserFromToken(c, tokenMaker, header) {
			return
		}
		c.Next()
	}
}

func setUserFromToken(c *gin.Context, tokenMaker token.Maker, header string) bool {
	subject, err := tokenMaker.VerifyToken(strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix)))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
		return false
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
		return false
	}
	c.Set(authUserIDKey, userID)
	return true
}

// AuthUserID returns the ID of the logged-in user, as stored by AuthMiddleware
// or OptionalAuthMiddleware. It returns false for anonymous requests.
func AuthUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(authUserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok
}
