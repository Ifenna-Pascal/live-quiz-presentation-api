package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/presentation/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// PresentationRouting returns the list of routers for domain presentation
// authMiddleware protects the routes that only the presentation owner may call.
func PresentationRouting(handler v1.PresentationHandler, authMiddleware gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{
			Method:      http.MethodPost,
			Path:        "/presentations",
			Handle:      handler.CreatePresentationHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware, middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id",
			Handle:      handler.GetPresentationHandler,
			MiddleWares: []gin.HandlerFunc{},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id/polls/current",
			Handle:      handler.GetCurrentPollHandler,
			MiddleWares: []gin.HandlerFunc{},
		},
		{
			Method:      http.MethodPut,
			Path:        "/presentations/:presentation_id/polls/current",
			Handle:      handler.UpdateCurrentPollHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware},
		},
		{
			Method:      http.MethodGet,
			Path:        "/users/me/presentations",
			Handle:      handler.ListMyPresentationsHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware},
		},
	}
}
