package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/users"
	"github.com/sirupsen/logrus"
)

type UserHandler interface {
	CreateUserHandler(c *gin.Context)
	LoginUserHandler(c *gin.Context)
	RefreshTokenHandler(c *gin.Context)
	LogoutHandler(c *gin.Context)
}

type userHandler struct {
	logger  *logrus.Logger
	useCase users.Usecase
}

func NewUserHandler(logger *logrus.Logger, usecase users.Usecase) UserHandler {
	return userHandler{logger: logger, useCase: usecase}
}

func clientInfo(c *gin.Context) users.ClientInfo {
	return users.ClientInfo{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
}

// CreateUserHandler godoc
//
//	@Summary	Sign up
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		user	body		model.CreateUserRequestDTO	true	"The new user"
//	@Success	201		{object}	model.CreateUserResponseDTO
//	@Failure	400		{object}	model.ErrorResponse	"Invalid body"
//	@Failure	409		{object}	model.ErrorResponse	"Email already registered"
//	@Failure	415		{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure	429		{object}	model.ErrorResponse	"Too many requests"
//	@Failure	500		{object}	model.ErrorResponse
//	@Router		/users [post]
func (server userHandler) CreateUserHandler(c *gin.Context) {
	var user model.CreateUserRequestDTO
	if err := c.ShouldBindJSON(&user); err != nil {
		respond.BadBody(c, err)
		return
	}
	created, err := server.useCase.CreateUser(c.Request.Context(), &user)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

// LoginUserHandler godoc
//
//	@Summary		Log in
//	@Description	Returns a short-lived access token and a single-use refresh token.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		model.LoginUserRequestDTO	true	"Email and password"
//	@Success		200			{object}	model.LoginUserResponseDTO
//	@Failure		400			{object}	model.ErrorResponse	"Invalid body"
//	@Failure		401			{object}	model.ErrorResponse	"Wrong email or password"
//	@Failure		415			{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		429			{object}	model.ErrorResponse	"Too many requests"
//	@Failure		500			{object}	model.ErrorResponse
//	@Router			/users/login [post]
func (server userHandler) LoginUserHandler(c *gin.Context) {
	var user model.LoginUserRequestDTO
	if err := c.ShouldBindJSON(&user); err != nil {
		respond.BadBody(c, err)
		return
	}
	userInfo, err := server.useCase.LoginUser(c.Request.Context(), &user, clientInfo(c))
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, userInfo)
}

// RefreshTokenHandler godoc
//
//	@Summary		Refresh the access token
//	@Description	Exchanges a refresh token for a new access token and a new refresh token. The old refresh token stops working. Presenting an already-used refresh token ends every session of that user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		model.RefreshTokenRequestDTO	true	"The refresh token"
//	@Success		200		{object}	model.LoginUserResponseDTO
//	@Failure		400		{object}	model.ErrorResponse	"Invalid body"
//	@Failure		401		{object}	model.ErrorResponse	"Refresh token invalid, expired or already used"
//	@Failure		415		{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/users/token/refresh [post]
func (server userHandler) RefreshTokenHandler(c *gin.Context) {
	var body model.RefreshTokenRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	userInfo, err := server.useCase.RefreshSession(c.Request.Context(), body.RefreshToken, clientInfo(c))
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.JSON(http.StatusOK, userInfo)
}

// LogoutHandler godoc
//
//	@Summary	Log out
//	@Tags		users
//	@Accept		json
//	@Param		body	body	model.RefreshTokenRequestDTO	true	"The refresh token to revoke"
//	@Success	204		"Logged out"
//	@Failure	400		{object}	model.ErrorResponse	"Invalid body"
//	@Failure	415		{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure	500		{object}	model.ErrorResponse
//	@Router		/users/logout [post]
func (server userHandler) LogoutHandler(c *gin.Context) {
	var body model.RefreshTokenRequestDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.BadBody(c, err)
		return
	}
	if err := server.useCase.Logout(c.Request.Context(), body.RefreshToken); err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}
