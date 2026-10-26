package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/presentation"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type PresentationHandler interface {
	CreatePresentationHandler(c *gin.Context)
	GetPresentationHandler(c *gin.Context)
	UpdatePresentationHandler(c *gin.Context)
	DeletePresentationHandler(c *gin.Context)
	UpdateCurrentPollHandler(c *gin.Context)
	PreviousPollHandler(c *gin.Context)
	RevealCurrentPollHandler(c *gin.Context)
	FinishPresentationHandler(c *gin.Context)
	GetCurrentPollHandler(c *gin.Context)
	ListMyPresentationsHandler(c *gin.Context)
}

type presentationHandler struct {
	logger  *logrus.Logger
	useCase presentation.Usecase
}

func NewPresentationHandler(logger *logrus.Logger, usecase presentation.Usecase) PresentationHandler {
	return presentationHandler{logger: logger, useCase: usecase}
}

// CreatePresentationHandler godoc
//
//	@Summary		Create a presentation
//	@Description	1-50 polls, each with a question and 2-6 options with unique keys. Mark right answers with `correct: true`.
//	@Tags			presentations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation	body		model.CreatePresentionRequestDTO	true	"Title and polls"
//	@Success		201				{object}	model.CreatePresentationResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid body or rule broken"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations [post]
func (server presentationHandler) CreatePresentationHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	var body model.CreatePresentionRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	id, err := server.useCase.CreatePresentation(c.Request.Context(), userID, &body)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusCreated, model.CreatePresentationResponse{PresentationID: id.String()})
}

// GetPresentationHandler godoc
//
//	@Summary		Get a presentation
//	@Description	Public. With the owner's access token the options include `correct` and `is_owner` is true; otherwise answers are only shown for revealed polls.
//	@Tags			presentations
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.PresentionResponseDTO
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"An Authorization header was sent but is invalid"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id} [get]
func (server presentationHandler) GetPresentationHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	var viewer *uuid.UUID
	if userID, ok := middleware.AuthUserID(c); ok {
		viewer = &userID
	}
	pres, err := server.useCase.GetPresentation(c.Request.Context(), viewer, presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, pres)
}

// UpdatePresentationHandler godoc
//
//	@Summary		Edit a presentation
//	@Description	Replaces the title and every poll. Only before the presentation starts. Owner only.
//	@Tags			presentations
//	@Accept			json
//	@Security		BearerAuth
//	@Param			presentation_id	path	string								true	"Presentation ID"	Format(uuid)
//	@Param			presentation	body	model.CreatePresentionRequestDTO	true	"Title and polls"
//	@Success		204				"Saved"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, body or rule broken"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		409				{object}	model.ErrorResponse	"Already started"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id} [put]
func (server presentationHandler) UpdatePresentationHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	var body model.CreatePresentionRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	if err := server.useCase.UpdatePresentation(c.Request.Context(), userID, presentationID, &body); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeletePresentationHandler godoc
//
//	@Summary		Delete a presentation
//	@Description	Deletes the presentation with its polls, players, votes and chat. Owner only.
//	@Tags			presentations
//	@Security		BearerAuth
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Success		204				"Deleted"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id} [delete]
func (server presentationHandler) DeletePresentationHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	if err := server.useCase.DeletePresentation(c.Request.Context(), userID, presentationID); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UpdateCurrentPollHandler godoc
//
//	@Summary		Show the next poll
//	@Description	Owner only. Returns the new poll, with correct answers.
//	@Tags			presentations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, or no more polls"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current [put]
func (server presentationHandler) UpdateCurrentPollHandler(c *gin.Context) {
	server.ownerPollAction(c, server.useCase.UpdateCurrentPoll)
}

// PreviousPollHandler godoc
//
//	@Summary		Show the previous poll
//	@Description	Owner only. Never goes before poll 1.
//	@Tags			presentations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, or already on the first poll"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/previous [put]
func (server presentationHandler) PreviousPollHandler(c *gin.Context) {
	server.ownerPollAction(c, server.useCase.PreviousPoll)
}

// RevealCurrentPollHandler godoc
//
//	@Summary		Reveal the answer of the poll on screen
//	@Description	Owner only. Closes voting on the poll, shows its correct answer to everyone and adds it to the leaderboard.
//	@Tags			presentations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, or no poll on screen"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		409				{object}	model.ErrorResponse	"Not started"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current/reveal [post]
func (server presentationHandler) RevealCurrentPollHandler(c *gin.Context) {
	server.ownerPollAction(c, server.useCase.RevealCurrentPoll)
}

func (server presentationHandler) ownerPollAction(c *gin.Context, action func(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error)) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	poll, err := action(c.Request.Context(), userID, presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, poll)
}

// FinishPresentationHandler godoc
//
//	@Summary		Finish the presentation
//	@Description	Owner only. Closes voting and counts every shown poll on the leaderboard.
//	@Tags			presentations
//	@Security		BearerAuth
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Success		204				"Finished"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		409				{object}	model.ErrorResponse	"Not started, or already finished"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/finish [post]
func (server presentationHandler) FinishPresentationHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	if err := server.useCase.FinishPresentation(c.Request.Context(), userID, presentationID); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetCurrentPollHandler godoc
//
//	@Summary		Get the poll on screen
//	@Description	Public. `correct` flags appear once the poll is revealed.
//	@Tags			presentations
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, or no poll on screen yet"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current [get]
func (server presentationHandler) GetCurrentPollHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	poll, err := server.useCase.GetCurrentPoll(c.Request.Context(), presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, poll)
}

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// ListMyPresentationsHandler godoc
//
//	@Summary		List my presentations
//	@Description	Returns the presentations owned by the logged-in user, newest first.
//	@Tags			presentations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Page size (1-100)"					default(20)
//	@Param			offset	query		int	false	"Number of presentations to skip"	default(0)
//	@Success		200		{array}		model.MyPresentationResponse
//	@Failure		400		{object}	model.ErrorResponse	"Invalid `limit` or `offset`"
//	@Failure		401		{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/users/me/presentations [get]
func (server presentationHandler) ListMyPresentationsHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 || limit > maxPageLimit {
		respond.Message(c, http.StatusBadRequest, "`limit` must be a number from 1 to 100.")
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		respond.Message(c, http.StatusBadRequest, "`offset` must be a number of 0 or more.")
		return
	}
	presentations, err := server.useCase.ListMyPresentations(c.Request.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, presentations)
}
