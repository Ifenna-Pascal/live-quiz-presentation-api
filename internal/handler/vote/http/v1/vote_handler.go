package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
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

func NewVoteHandler(logger *logrus.Logger, useCase vote.Usecase) voteHandler {
	return voteHandler{
		logger:  logger,
		useCase: useCase,
	}
}

// CreateVoteHandler godoc
//
//	@Summary		Vote on the current poll
//	@Description	Records one vote of a registered player for an option of the poll currently displayed. `poll_id` must be the ID of that poll, and the presentation must have started.
//	@Tags			votes
//	@Accept			json
//	@Produce		json
//	@Param			presentation_id	path	string						true	"Presentation ID"	Format(uuid)
//	@Param			vote			body	model.CreateVoteRequestDTO	true	"The vote"
//	@Success		204				"Vote recorded"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`, invalid body, `poll_id` is not the current poll, no poll displayed yet, or unknown option key"
//	@Failure		403				{object}	model.ErrorResponse	"`player_id` is not registered in this presentation, or the presentation has not started"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation or poll not found"
//	@Failure		409				{object}	model.ErrorResponse	"The player already voted on this poll"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500				{object}	model.ErrorResponse	"Any other failure"
//	@Router			/presentations/{presentation_id}/polls/current/votes [post]
func (server voteHandler) CreateVoteHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var voteParam model.CreateVoteRequestDTO

	if err := c.ShouldBindJSON(&voteParam); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = server.useCase.CreateVote(context.Background(), presentationID, voteParam)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Either `presentation_id` or `poll_id` not found"})
			return
		case model.ErrIDMismatch:
			c.JSON(http.StatusBadRequest, gin.H{"error": "The `poll_id` in the request body doesn't match the current poll."})
			return
		case model.ErrNoPollDisplayed:
			c.JSON(http.StatusBadRequest, gin.H{"error": "No poll is displayed yet."})
			return
		case model.ErrInvalidOption:
			c.JSON(http.StatusBadRequest, gin.H{"error": "The `key` is not an option of this poll."})
			return
		case model.ErrPlayerNotRegistered:
			c.JSON(http.StatusForbidden, gin.H{"error": "The player is not registered in this presentation, or the presentation has not started."})
			return
		case model.ErrAlreadyVoted:
			c.JSON(http.StatusConflict, gin.H{"error": "The player already voted on this poll."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusNoContent, "")
}

// GetPollVotesHandler godoc
//
//	@Summary		List the votes of a poll
//	@Description	Returns every vote cast on the given poll, ordered by option key. Any poll of the presentation can be read, including past ones.
//	@Tags			votes
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Param			poll_id			path		string	true	"Poll ID"			Format(uuid)
//	@Success		200				{array}		model.VoteResponse
//	@Failure		400				"An invalid UUID in the path (empty body)"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation or poll not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/{poll_id}/votes [get]
func (server voteHandler) GetPollVotesHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	pollID, err := uuid.Parse(c.Param("poll_id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	votes, err := server.useCase.GetPollVotes(context.Background(), presentationID, pollID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Either `presentation_id` or `poll_id` not found"})
			return
		case model.ErrIDMismatch:
			c.JSON(http.StatusBadRequest, gin.H{"error": "The `poll_id` in the request body doesn't match the current poll."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, votes)
}

// ListPresentationVotesHandler godoc
//
//	@Summary		List all votes of a presentation
//	@Description	Returns every vote of every poll, ordered by poll, with whether each vote was correct. Only the owner of the presentation can do this.
//	@Tags			votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{array}		model.PresentationVoteResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"The logged-in user does not own the presentation"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/votes [get]
func (server voteHandler) ListPresentationVotesHandler(c *gin.Context) {
	userID, presentationID, ok := ownerRequest(c)
	if !ok {
		return
	}
	votes, err := server.useCase.ListPresentationVotes(context.Background(), userID, presentationID)
	if err != nil {
		writeOwnerError(c, err)
		return
	}
	c.JSON(http.StatusOK, votes)
}

// GetLeaderboardHandler godoc
//
//	@Summary		Get the leaderboard
//	@Description	Ranks every registered player by the number of polls answered correctly (one point per correct vote). Polls without a correct option are not scored. Players with the same score share a rank. Only the owner of the presentation can do this.
//	@Tags			votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.LeaderboardResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"The logged-in user does not own the presentation"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/leaderboard [get]
func (server voteHandler) GetLeaderboardHandler(c *gin.Context) {
	userID, presentationID, ok := ownerRequest(c)
	if !ok {
		return
	}
	leaderboard, err := server.useCase.GetLeaderboard(context.Background(), userID, presentationID)
	if err != nil {
		writeOwnerError(c, err)
		return
	}
	c.JSON(http.StatusOK, leaderboard)
}

// ownerRequest reads the logged-in user and the presentation_id path
// parameter. On failure it writes the error response and returns false.
func ownerRequest(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := middleware.AuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not logged in."})
		return uuid.Nil, uuid.Nil, false
	}
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return uuid.Nil, uuid.Nil, false
	}
	return userID, presentationID, true
}

// writeOwnerError writes the response for an error of an owner-only use case.
func writeOwnerError(c *gin.Context, err error) {
	switch err {
	case model.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "The presentation was not found."})
	case model.ErrForbidden:
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the owner of the presentation can do this."})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
