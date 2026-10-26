package routing

import (
	"net/http"

	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/realtime/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// RealtimeRouting returns the Server-Sent Events stream route.
func RealtimeRouting(handler v1.RealtimeHandler) []routers.Route {
	return []routers.Route{
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/events", Handle: handler.StreamHandler},
	}
}
