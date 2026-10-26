package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/player"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type PlayerHandler interface {
	RegisterPlayerHandler(c *gin.Context)
	ListPlayersHandler(c *gin.Context)
	StartPresentationHandler(c *gin.Context)
	RemovePlayerHandler(c *gin.Context)
}

type playerHandler struct {
	logger  *logrus.Logger
	useCase player.Usecase
}

func NewPlayerHandler(logger *logrus.Logger, useCase player.Usecase) PlayerHandler {
	return playerHandler{logger: logger, useCase: useCase}
}

// RegisterPlayerHandler godoc
//
//	@Summary		Join a presentation
//	@Description	Registers a player while the presentation has not started. The response contains `player_token`, a secret the player must send to vote and chat. It is shown only once.
//	@Tags			players
//	@Accept			json
//	@Produce		json
//	@Param			presentation_id	path		string								true	"Presentation ID"	Format(uuid)
//	@Param			player			body		model.RegisterPlayerToPresentation	true	"The player"
//	@Success		201				{object}	model.RegisteredPlayerResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, body, or name"
//	@Failure		409				{object}	model.ErrorResponse	"Name taken, or presentation missing or already started"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		429				{object}	model.ErrorResponse	"Too many requests"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/players [post]
func (server playerHandler) RegisterPlayerHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	var body model.RegisterPlayerToPresentation
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	registered, err := server.useCase.RegisterPlayer(c.Request.Context(), presentationID, body)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusCreated, registered)
}

// ListPlayersHandler godoc
//
//	@Summary	List players
//	@Tags		players
//	@Produce	json
//	@Param		presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Success	200				{array}		model.CreatedPresentationPlayerResponse
//	@Failure	400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure	404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure	500				{object}	model.ErrorResponse
//	@Router		/presentations/{presentation_id}/players [get]
func (server playerHandler) ListPlayersHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	players, err := server.useCase.ListPlayers(c.Request.Context(), presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, players)
}

// StartPresentationHandler godoc
//
//	@Summary		Start the presentation
//	@Description	Closes registration and shows the first poll. Owner only.
//	@Tags			players
//	@Security		BearerAuth
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Success		204				"Started"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		409				{object}	model.ErrorResponse	"Already started"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/start [post]
func (server playerHandler) StartPresentationHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	if err := server.useCase.StartPresentation(c.Request.Context(), userID, presentationID); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RemovePlayerHandler godoc
//
//	@Summary		Remove a player
//	@Description	Deletes the player, their votes and their chat messages. Owner only.
//	@Tags			players
//	@Security		BearerAuth
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Param			player_id		path	string	true	"Player ID"			Format(uuid)
//	@Success		204				"Removed"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		401				{object}	model.ErrorResponse	"Missing, invalid or expired access token"
//	@Failure		403				{object}	model.ErrorResponse	"Not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation or player not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/players/{player_id} [delete]
func (server playerHandler) RemovePlayerHandler(c *gin.Context) {
	userID, _ := middleware.AuthUserID(c)
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	playerID, ok := respond.UUIDParam(c, "player_id")
	if !ok {
		return
	}
	if err := server.useCase.RemovePlayer(c.Request.Context(), userID, presentationID, playerID); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}
