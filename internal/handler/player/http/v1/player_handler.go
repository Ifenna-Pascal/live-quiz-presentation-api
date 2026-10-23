package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/player"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type PlayerHandler interface {
	RegisterPlayerHandler(c *gin.Context)
	ListPlayersHandler(c *gin.Context)
	StartPresentationHandler(c *gin.Context)
}

type playerHandler struct {
	logger  *logrus.Logger
	useCase player.Usecase
}

func NewPlayerHandler(logger *logrus.Logger, useCase player.Usecase) PlayerHandler {
	return playerHandler{
		logger:  logger,
		useCase: useCase,
	}
}

// RegisterPlayerHandler godoc
//
//	@Summary		Register a player
//	@Description	Registers a player in a presentation. Registration is only open until the presentation starts, and a name can be used only once per presentation.
//	@Tags			players
//	@Accept			json
//	@Produce		json
//	@Param			presentation_id	path		string								true	"Presentation ID"	Format(uuid)
//	@Param			player			body		model.RegisterPlayerToPresentation	true	"The player"
//	@Success		201				{object}	model.CreatedPresentationPlayerResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`, invalid body, or empty `player_name`"
//	@Failure		409				{object}	model.ErrorResponse	"Presentation not found or already started, or `player_name` already taken"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/players [post]
func (server playerHandler) RegisterPlayerHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var playerParam model.RegisterPlayerToPresentation
	if err := c.ShouldBindJSON(&playerParam); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	player, err := server.useCase.RegisterPlayer(context.Background(), presentationID, playerParam)
	if err != nil {
		switch err {
		case model.ErrInvalidPlayerName:
			c.JSON(http.StatusBadRequest, gin.H{"error": "The `player_name` cannot be empty."})
			return
		case model.ErrRegistrationClosed:
			c.JSON(http.StatusConflict, gin.H{"error": "The presentation does not exist or has already started."})
			return
		case model.ErrNameTaken:
			c.JSON(http.StatusConflict, gin.H{"error": "The `player_name` is already taken in this presentation."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, player)
}

// ListPlayersHandler godoc
//
//	@Summary		List the players of a presentation
//	@Description	Returns every player registered in the presentation, in the order they registered.
//	@Tags			players
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success		200				{array}		model.CreatedPresentationPlayerResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/players [get]
func (server playerHandler) ListPlayersHandler(c *gin.Context) {
	presentationID, err := uuid.Parse(c.Param("presentation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	players, err := server.useCase.ListPlayers(context.Background(), presentationID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "The presentation was not found."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, players)
}

// StartPresentationHandler godoc
//
//	@Summary		Start a presentation
//	@Description	Starts the presentation: closes player registration, opens voting and displays the first poll. Only the owner of the presentation can do this.
//	@Tags			players
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Success		204				"Presentation started and first poll displayed"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid `presentation_id`"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"The logged-in user does not own the presentation"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		409				{object}	model.ErrorResponse	"Presentation already started"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/start [post]
func (server playerHandler) StartPresentationHandler(c *gin.Context) {
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

	err = server.useCase.StartPresentation(context.Background(), userID, presentationID)
	if err != nil {
		switch err {
		case model.ErrForbidden:
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the owner of the presentation can do this."})
			return
		case model.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "The presentation was not found."})
			return
		case model.ErrPresentationAlreadyStarted:
			c.JSON(http.StatusConflict, gin.H{"error": "The presentation has already started."})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.Status(http.StatusNoContent)
}
