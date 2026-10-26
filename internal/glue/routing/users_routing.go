package routing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/users/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// UserRouting returns the routes for accounts and sessions.
func UserRouting(handler v1.UserHandler) []routers.Route {
	return []routers.Route{
		{
			Method:      http.MethodPost,
			Path:        "/users",
			Handle:      handler.CreateUserHandler,
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(20, time.Minute), middleware.ContentTypeChecker()},
		},
		{
			Method: http.MethodPost,
			Path:   "/users/login",
			Handle: handler.LoginUserHandler,
			// Slows down password guessing.
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(20, time.Minute), middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodPost,
			Path:        "/users/token/refresh",
			Handle:      handler.RefreshTokenHandler,
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(60, time.Minute), middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodPost,
			Path:        "/users/logout",
			Handle:      handler.LogoutHandler,
			MiddleWares: []gin.HandlerFunc{middleware.ContentTypeChecker()},
		},
	}
}
