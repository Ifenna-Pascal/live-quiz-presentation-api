package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/users"
	"github.com/sirupsen/logrus"
)

type UserHandler interface {
	CreateUserHandler(c *gin.Context)
	LoginUserHandler(c *gin.Context)
}

type userHandler struct {
	logger  *logrus.Logger
	useCase users.Usecase
}

func NewUserHandler(logger *logrus.Logger, usecase users.Usecase) UserHandler {
	return userHandler{
		logger:  logger,
		useCase: usecase,
	}
}

// CreateUserHandler godoc
//
//	@Summary		Create a user
//	@Description	Creates a user. The password is hashed before it is stored and is never returned.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		model.CreateUserRequestDTO	true	"The new user"
//	@Success		201		{object}	model.CreateUserResponseDTO
//	@Failure		400		{object}	model.ErrorResponse	"Body is missing mandatory fields, the email is invalid, or the password is shorter than 8 characters"
//	@Failure		409		{object}	model.ErrorResponse	"A user with this email already exists"
//	@Failure		415		{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/users [post]
func (server userHandler) CreateUserHandler(c *gin.Context) {
	var user model.CreateUserRequestDTO
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := server.useCase.CreateUser(context.Background(), &user)
	if err != nil {
		switch err {
		case model.ErrEmailTaken:
			c.JSON(http.StatusConflict, gin.H{"error": "a user with this email already exists"})
			return
		default:
			server.logger.Error("create user: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			return
		}
	}

	c.JSON(http.StatusCreated, created)
}

// LoginUserHandler godoc
//
//	@Summary		Log in a user
//	@Description	Checks the email and password and returns the user with a signed access token.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		model.LoginUserRequestDTO	true	"Email and password"
//	@Success		200			{object}	model.LoginUserResponseDTO
//	@Failure		400			{object}	model.ErrorResponse	"Body is missing mandatory fields or the email is invalid"
//	@Failure		401			{object}	model.ErrorResponse	"The email or password is wrong"
//	@Failure		415			{object}	model.ErrorResponse	"Content-Type is not application/json"
//	@Failure		500			{object}	model.ErrorResponse
//	@Router			/users/login [post]
func (server userHandler) LoginUserHandler(c *gin.Context) {
	var user model.LoginUserRequestDTO
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userInfo, err := server.useCase.LoginUser(context.Background(), &user)
	if err != nil {
		switch err {
		case model.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		default:
			server.logger.Error("login user: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
			return
		}
	}

	c.JSON(http.StatusOK, userInfo)
}
