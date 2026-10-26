package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/vote"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type VoteHandler interface {
	CreateVoteHandler(c *gin.Context)
	GetPollVotesHandler(c *gin.Context)
	ListPresentationVotesHandler(c *gin.Context)
	GetLeaderboardHandler(c *gin.Context)
}

type voteHandler struct {
	logger  *logrus.Logger
	useCase vote.Usecase
}

func NewVoteHandler(logger *logrus.Logger, useCase vote.Usecase) VoteHandler {
	return voteHandler{logger: logger, useCase: useCase}
}

// CreateVoteHandler godoc
//
//	@Summary		Answer the poll on screen
//	@Description	The player proves who they are with the `player_token` they got at registration.
//	@Description	Timed: answers are accepted until `closes_at`. The first correct answer wins the poll and closes it; answers are handled one at a time, so exactly one player can win.
//	@Description	On scored polls a wrong answer can be followed by another option (each option once). On opinion polls each player answers once.
//	@Tags			votes
//	@Accept			json
//	@Produce		json
//	@Param			presentation_id	path	string						true	"Presentation ID"	Format(uuid)
//	@Param			vote			body	model.CreateVoteRequestDTO	true	"The vote"
//	@Success		200				{object}	model.VoteResult	"Answer recorded: correct or not, and whether it won the poll"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id or body, not the poll on screen, no poll on screen, or unknown option"
//	@Failure		403				{object}	model.ErrorResponse	"Player not registered, wrong player token, or not started"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation or poll not found"
//	@Failure		409				{object}	model.ErrorResponse	"Option already tried, someone already won, time is up, or voting closed"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		429				{object}	model.ErrorResponse	"Too many requests"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current/votes [post]
func (server voteHandler) CreateVoteHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	var body model.CreateVoteRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	result, err := server.useCase.CreateVote(c.Request.Context(), presentationID, body)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetPollVotesHandler godoc
//
//	@Summary	List the votes of a poll
//	@Tags		votes
//	@Produce	json
//	@Param		presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Param		poll_id			path		string	true	"Poll ID"			Format(uuid)
//	@Success	200				{array}		model.VoteResponse
//	@Failure	400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure	404				{object}	model.ErrorResponse	"Presentation or poll not found"
//	@Failure	500				{object}	model.ErrorResponse
//	@Router		/presentations/{presentation_id}/polls/{poll_id}/votes [get]
func (server voteHandler) GetPollVotesHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	pollID, ok := respond.UUIDParam(c, "poll_id")
	if !ok {
		return
	}
	votes, err := server.useCase.GetPollVotes(c.Request.Context(), presentationID, pollID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, votes)
}

// ListPresentationVotesHandler godoc
//
//	@Summary		List all votes of a presentation
//	@Description	Every vote of every poll, with whether it was correct. Owner only.
//	@Tags			votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{array}		model.PresentationVoteResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/votes [get]
func (server voteHandler) ListPresentationVotesHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	votes, err := server.useCase.ListPresentationVotes(c.Request.Context(), userID, presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, votes)
}

// GetLeaderboardHandler godoc
//
//	@Summary		Get the leaderboard
//	@Description	Public. One point per correct answer; a poll only counts once it is revealed (or the presentation has finished). Ties share a rank.
//	@Tags			votes
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.LeaderboardResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/leaderboard [get]
func (server voteHandler) GetLeaderboardHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	leaderboard, err := server.useCase.GetLeaderboard(c.Request.Context(), presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, leaderboard)
}
