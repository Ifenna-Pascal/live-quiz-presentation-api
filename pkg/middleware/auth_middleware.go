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

		subject, err := tokenMaker.VerifyToken(strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix)))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
			return
		}

		userID, err := uuid.Parse(subject)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token."})
			return
		}

		c.Set(authUserIDKey, userID)
		c.Next()
	}
}

// AuthUserID returns the ID of the logged-in user, as stored by AuthMiddleware.
// It returns false if the route is not behind AuthMiddleware.
func AuthUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(authUserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok
}
