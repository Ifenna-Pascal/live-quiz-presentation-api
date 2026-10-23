package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/vote/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// VoteRouting returns the list of routers for domain vote
// The vote history and the leaderboard are owner-only and need authMiddleware.
func VoteRouting(handler v1.VoteHandler, authMiddleware gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{
			Method:      http.MethodPost,
			Path:        "/presentations/:presentation_id/polls/current/votes",
			Handle:      handler.CreateVoteHandler,
			MiddleWares: []gin.HandlerFunc{middleware.ContentTypeChecker()},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id/polls/:poll_id/votes",
			Handle:      handler.GetPollVotesHandler,
			MiddleWares: []gin.HandlerFunc{},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id/votes",
			Handle:      handler.ListPresentationVotesHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware},
		},
		{
			Method:      http.MethodGet,
			Path:        "/presentations/:presentation_id/leaderboard",
			Handle:      handler.GetLeaderboardHandler,
			MiddleWares: []gin.HandlerFunc{authMiddleware},
		},
	}
}
