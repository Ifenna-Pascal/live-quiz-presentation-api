package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/player/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// PlayerRouting returns the list of routers for domain player.
// Registering and listing players are public; starting needs authMiddleware.
func PlayerRouting(handler v1.PlayerHandler, authMiddleware gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{
			Method:      http.MethodPost,
			Path:        "/presentations/:presentation_id/players",
			Handle:      handler.RegisterPlayerHandler,
			MiddleWares: []gin.HandlerFunc{middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id/players",
			Handle:      handler.ListPlayersHandler,
			MiddleWares: []gin.HandlerFunc{},
		},
		{
			Method:      http.MethodPost,
			Path:        "/presentations/:presentation_id/start",
			Handle:      handler.StartPresentationHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware},
		},
	}
}
