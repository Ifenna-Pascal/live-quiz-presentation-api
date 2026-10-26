package routing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/chat/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// ChatRouting returns the chat routes. Reading is public; sending needs a
// player token or the owner's access token.
func ChatRouting(handler v1.ChatHandler, optionalAuth gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/messages", Handle: handler.ListMessagesHandler},
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/messages", Handle: handler.SendMessageHandler,
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(240, time.Minute), optionalAuth, middleware.ContentTypeChecker()}},
	}
}
