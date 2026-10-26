package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/chat"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/sirupsen/logrus"
)

type ChatHandler interface {
	SendMessageHandler(c *gin.Context)
	ListMessagesHandler(c *gin.Context)
}

type chatHandler struct {
	logger  *logrus.Logger
	useCase chat.Usecase
}

func NewChatHandler(logger *logrus.Logger, useCase chat.Usecase) ChatHandler {
	return chatHandler{logger: logger, useCase: useCase}
}

// SendMessageHandler godoc
//
//	@Summary		Send a chat message
//	@Description	Players send `player_id` and `player_token`. The host sends neither and authenticates with the access token.
//	@Tags			chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			presentation_id	path		string							true	"Presentation ID"	Format(uuid)
//	@Param			message			body		model.CreateMessageRequestDTO	true	"The message (1-500 characters)"
//	@Success		201				{object}	model.MessageResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id or body, or empty / too long message"
//	@Failure		401				{object}	model.ErrorResponse	"Authorization header sent but invalid"
//	@Failure		403				{object}	model.ErrorResponse	"Wrong player id or token, or not the owner"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		415				{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		429				{object}	model.ErrorResponse	"Too many requests"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/messages [post]
func (server chatHandler) SendMessageHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	var body model.CreateMessageRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	var userID *uuid.UUID
	if id, ok := middleware.AuthUserID(c); ok {
		userID = &id
	}
	msg, err := server.useCase.SendMessage(c.Request.Context(), presentationID, userID, body)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

// ListMessagesHandler godoc
//
//	@Summary		List chat messages
//	@Description	Oldest first. Without `after`: the newest `limit` messages. With `after` (RFC 3339): the messages sent after that time.
//	@Tags			chat
//	@Produce		json
//	@Param			presentation_id	path		string	true	"Presentation ID"	Format(uuid)
//	@Param			after			query		string	false	"Only messages after this time"	Format(date-time)
//	@Param			limit			query		int		false	"Max messages (1-200)"				default(100)
//	@Success		200				{array}		model.MessageResponse
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id, `after` or `limit`"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/presentations/{presentation_id}/messages [get]
func (server chatHandler) ListMessagesHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(chat.DefaultLimit)))
	if err != nil || limit < 1 || limit > chat.MaxLimit {
		respond.Message(c, http.StatusBadRequest, "`limit` must be a number from 1 to 200.")
		return
	}
	var after *time.Time
	if raw := c.Query("after"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			respond.Message(c, http.StatusBadRequest, "`after` must be an RFC 3339 time.")
			return
		}
		after = &t
	}
	messages, err := server.useCase.ListMessages(c.Request.Context(), presentationID, after, int32(limit))
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, messages)
}
