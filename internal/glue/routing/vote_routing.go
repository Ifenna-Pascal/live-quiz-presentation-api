package routing

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/vote/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

// VoteRouting returns the routes for votes and the leaderboard. The full
// vote history is owner-only; the leaderboard is public.
func VoteRouting(handler v1.VoteHandler, auth gin.HandlerFunc) []routers.Route {
	return []routers.Route{
		{Method: http.MethodPost, Path: "/presentations/:presentation_id/polls/current/votes", Handle: handler.CreateVoteHandler,
			MiddleWares: []gin.HandlerFunc{middleware.RateLimit(600, time.Minute), middleware.ContentTypeChecker()}},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/polls/:poll_id/votes", Handle: handler.GetPollVotesHandler},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/votes", Handle: handler.ListPresentationVotesHandler,
			MiddleWares: []gin.HandlerFunc{auth}},
		{Method: http.MethodGet, Path: "/presentations/:presentation_id/leaderboard", Handle: handler.GetLeaderboardHandler},
	}
}
