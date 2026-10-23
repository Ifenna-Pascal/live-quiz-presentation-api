package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/presentation"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type PresentationHandler interface {
	CreatePresentationHandler(c *gin.Context)
	GetPresentationHandler(c *gin.Context)
	UpdateCurrentPollHandler(c *gin.Context)
	GetCurrentPollHandler(c *gin.Context)
	ListMyPresentationsHandler(c *gin.Context)
}

type presentationHandler struct {
	logger  *logrus.Logger
	useCase presentation.Usecase
}

func NewPresentationHandler(logger *logrus.Logger, usecase presentation.Usecase) PresentationHandler {
	return presentationHandler{
		logger:  logger,
		useCase: usecase,
	}
}

// CreatePresentationHandler godoc
//
//	@Summary		Create a presentation
//	@Description	Creates a presentation owned by the logged-in user, together with its polls and their options. No poll is displayed yet: `current_poll_index` starts at 0 until the first `PUT .../polls/current`.
//	@Tags			presentations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation	body		model.CreatePresentionRequestDTO	true	"The polls of the presentation, in display order"
//	@Success		201				{object}	model.CreatePresentationResponse
//	@Failure		400				{string}	string				"Body is missing mandatory fields, has a wrong type, or contains no polls"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Router			/presentations [post]
func (server presentationHandler) CreatePresentationHandler(c *gin.Context) {
	userID, ok := middleware.AuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not logged in."})
		return
	}

	var presenation model.CreatePresentionRequestDTO
	if err := c.ShouldBindJSON(&presenation); err != nil {
		c.JSON(http.StatusBadRequest, "Mandatory body parameters missing or have incorrect type")
		return
	}

	if len(presenation.Polls) == 0 {
		c.JSON(http.StatusBadRequest, "no polls found")
		return
	}

	presID, err := server.useCase.CreatePresentation(context.Background(), userID, &presenation)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"presentation_id": presID,
	})
}

// GetPresentationHandler godoc
//
//	@Summary		Get a presentation
//	@Description	Returns the index of the poll currently displayed, whether the presentation has started (`has_started` is false while players can still register), and all polls of the presentation. Correct answers are never included.
//	@Tags			presentations
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.PresentionResponseDTO{polls=[]model.Poll}
//	@Failure		400				{object}	model.ErrorResponse	"`presentation_id` is not a valid UUID"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id} [get]
func (server presentationHandler) GetPresentationHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	presentationWithPoll, err := server.useCase.GetPresentation(context.Background(), presentationID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "there is no presentation with the provided `presentation_id`"})
			return
		case model.ErrNoPollDisplayed:
			c.JSON(http.StatusBadRequest, gin.H{"error": "there are no polls currently displayed"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	c.JSON(http.StatusOK, presentationWithPoll)
}

// UpdateCurrentPollHandler godoc
//
//	@Summary		Display the next poll
//	@Description	Moves the presentation forward by one poll and returns the poll that is now displayed. Only the owner of the presentation can do this.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"`presentation_id` is not a valid UUID, or the presentation ran out of polls"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"The logged-in user does not own the presentation"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current [put]
func (server presentationHandler) UpdateCurrentPollHandler(c *gin.Context) {
	userID, ok := middleware.AuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not logged in."})
		return
	}

	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	currentPoll, err := server.useCase.UpdateCurrentPoll(context.Background(), userID, presentationID)
	if err != nil {
		switch err {
		case model.ErrForbidden:
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the owner of the presentation can do this."})
			return
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "there is no presentation with the provided `presentation_id`"})
			return
		case model.ErrRunOutOfIndex:
			// Already on the last poll - there is no "next".
			c.JSON(http.StatusBadRequest, gin.H{"error": "The presentation ran out of polls."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, currentPoll)
}

// GetCurrentPollHandler godoc
//
//	@Summary		Get the current poll
//	@Description	Returns the poll that is currently displayed.
//	@Tags			polls
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{object}	model.CurrentPoll
//	@Failure		400				{object}	model.ErrorResponse	"No poll is currently displayed. (An invalid `presentation_id` also gives 400, with a plain JSON string as body.)"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/polls/current [get]
func (server presentationHandler) GetCurrentPollHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// 2. CALL the core.
	currentPoll, err := server.useCase.GetCurrentPoll(context.Background(), presentationID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "there is no presentation with the provided `presentation_id`"})
			return
		case model.ErrNoPollDisplayed:
			c.JSON(http.StatusBadRequest, gin.H{"error": "there are no polls currently displayed"})
			return
		case model.ErrRunOutOfIndex:
			c.JSON(http.StatusBadRequest, gin.H{"error": "The presentation ran out of polls."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	c.JSON(http.StatusOK, currentPoll)
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
//	@Param			limit	query		int	false	"Page size (1-100)"	default(20)
//	@Param			offset	query		int	false	"Number of presentations to skip"	default(0)
//	@Success		200		{array}		model.MyPresentationResponse
//	@Failure		400		{object}	model.ErrorResponse	"Invalid `limit` or `offset`"
//	@Failure		401		{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/users/me/presentations [get]
func (server presentationHandler) ListMyPresentationsHandler(c *gin.Context) {
	userID, ok := middleware.AuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not logged in."})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 || limit > maxPageLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": "`limit` must be a number from 1 to 100."})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "`offset` must be a number of 0 or more."})
		return
	}

	presentations, err := server.useCase.ListMyPresentations(context.Background(), userID, int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, presentations)
}
