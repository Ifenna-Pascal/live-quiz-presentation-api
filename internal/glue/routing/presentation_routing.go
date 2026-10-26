package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/presentation/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// PresentationRouting returns the routes for presentations and their polls.
// auth protects owner-only routes; optionalAuth lets the owner see more.
func PresentationRouting(handler v1.PresentationHandler, auth, optionalAuth gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{Method: http.MethodPost, Path: "/presentations", Handle: handler.CreatePresentationHandler,
			MiddleWares: []gin.HandlerFunc{auth, middleware.ContentTypeChecker()}},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id", Handle: handler.GetPresentationHandler,
			MiddleWares: []gin.HandlerFunc{optionalAuth}},
		{Method: http.MethodPut, Path: "/presentations/:presentation_id", Handle: handler.UpdatePresentationHandler,
			MiddleWares: []gin.HandlerFunc{auth, middleware.ContentTypeChecker()}},
		{Method: http.MethodDelete, Path: "/presentations/:presentation_id", Handle: handler.DeletePresentationHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/polls/current", Handle: handler.GetCurrentPollHandler},
		{Method: http.MethodPut, Path: "/presentations/:presentation_id/polls/current", Handle: handler.UpdateCurrentPollHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodPut, Path: "/presentations/:presentation_id/polls/previous", Handle: handler.PreviousPollHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/polls/current/reveal", Handle: handler.RevealCurrentPollHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/finish", Handle: handler.FinishPresentationHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodGet, Path: "/users/me/presentations", Handle: handler.ListMyPresentationsHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
	}
}
