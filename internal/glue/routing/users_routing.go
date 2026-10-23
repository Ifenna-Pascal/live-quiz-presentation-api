package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/users/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// UserRouting returns the list of routers for domain users
func UserRouting(handler v1.UserHandler) []routers.Route {
	return []routers.Route{
		{
			Method:      http.MethodPost,
			Path:        "/users",
			Handle:      handler.CreateUserHandler,
			MiddleWares: []gin.HandlerFunc{middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodPost,
			Path:        "/users/login",
			Handle:      handler.LoginUserHandler,
			MiddleWares: []gin.HandlerFunc{middleware.ContentTypeChecker()},
		},
	}
}
