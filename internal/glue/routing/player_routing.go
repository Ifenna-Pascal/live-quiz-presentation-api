package routing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/player/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// PlayerRouting returns the routes for players. Joining and listing are
// public; starting and removing players are owner-only.
func PlayerRouting(handler v1.PlayerHandler, auth gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/players", Handle: handler.RegisterPlayerHandler,
			// A whole classroom can share one IP, so the limit is high.
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(120, time.Minute), middleware.ContentTypeChecker()}},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/players", Handle: handler.ListPlayersHandler},
		{Method: http.MethodDelete, Path: "/presentations/:presentation_id/players/:player_id", Handle: handler.RemovePlayerHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/start", Handle: handler.StartPresentationHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
	}
}
