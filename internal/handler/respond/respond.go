// Package respond writes errors in one shape: {"error": "<message>"}.
package respond

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/sirupsen/logrus"
)

type mapped struct {
	status  int
	message string
}

var known = map[error]mapped{
	model.ErrNotFound:                   {http.StatusNotFound, "Not found."},
	model.ErrIDMismatch:                 {http.StatusBadRequest, "The `poll_id` doesn't match the poll on screen."},
	model.ErrNoPollDisplayed:            {http.StatusBadRequest, "No poll is on screen yet."},
	model.ErrRunOutOfIndex:              {http.StatusBadRequest, "There is no poll in that direction."},
	model.ErrEmailTaken:                 {http.StatusConflict, "A user with this email already exists."},
	model.ErrInvalidCredentials:         {http.StatusUnauthorized, "Invalid email or password."},
	model.ErrPlayerNotRegistered:        {http.StatusForbidden, "The player is not registered here, the player token is wrong, or the presentation has not started."},
	model.ErrAlreadyVoted:               {http.StatusConflict, "You already chose this answer (opinion polls take one answer per player)."},
	model.ErrTimeUp:                     {http.StatusConflict, "Time's up for this question."},
	model.ErrPollAlreadyWon:             {http.StatusConflict, "Someone else answered first."},
	model.ErrRegistrationClosed:         {http.StatusConflict, "The presentation does not exist or has already started."},
	model.ErrNameTaken:                  {http.StatusConflict, "That name is already taken in this presentation."},
	model.ErrPresentationAlreadyStarted: {http.StatusConflict, "The presentation has already started."},
	model.ErrPresentationNotRunning:     {http.StatusConflict, "The presentation is not running (not started yet, or already finished)."},
	model.ErrInvalidPlayerName:          {http.StatusBadRequest, "The `player_name` cannot be empty."},
	model.ErrForbidden:                  {http.StatusForbidden, "Only the owner of the presentation can do this."},
	model.ErrInvalidOption:              {http.StatusBadRequest, "The `key` is not an option of this poll."},
	model.ErrVotingClosed:               {http.StatusConflict, "Voting on this poll is closed."},
	model.ErrInvalidPlayer:              {http.StatusForbidden, "The player id or player token is invalid."},
	model.ErrInvalidRefreshToken:        {http.StatusUnauthorized, "The refresh token is invalid or expired. Please log in again."},
}

// Error writes the status and message for err. Unknown errors are logged
// and answered with a generic 500 so internals never leak.
func Error(c *gin.Context, logger *logrus.Logger, err error) {
	var validation model.ValidationError
	if errors.As(err, &validation) {
		Message(c, http.StatusBadRequest, validation.Message)
		return
	}
	for target, m := range known {
		if errors.Is(err, target) {
			Message(c, m.status, m.message)
			return
		}
	}
	logger.WithField("path", c.FullPath()).Error(err)
	Message(c, http.StatusInternalServerError, "Something went wrong. Please try again.")
}

// Message writes {"error": message} and stops the handler chain.
func Message(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, model.ErrorResponse{Error: message})
}

// BadBody answers a body that is not valid JSON or misses required fields.
func BadBody(c *gin.Context, err error) {
	Message(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
}

// UUIDParam parses a path parameter, answering 400 when it is not a UUID.
func UUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		Message(c, http.StatusBadRequest, "`"+name+"` must be a UUID.")
		return uuid.Nil, false
	}
	return id, true
}
