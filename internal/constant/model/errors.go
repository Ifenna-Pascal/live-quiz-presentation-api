package model

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrIDMismatch      = errors.New("id mismached")
	ErrNoPollDisplayed = errors.New("not displayed")
	ErrRunOutOfIndex   = errors.New("poll run out of index")
	ErrEmailTaken      = errors.New("email already in use")
	// ErrInvalidCredentials covers both an unknown email and a wrong password,
	// so a caller cannot use login to find out which emails are registered.
	ErrInvalidCredentials         = errors.New("invalid email or password")
	ErrPlayerNotRegistered        = errors.New("player not registered or presentation not started")
	ErrAlreadyVoted               = errors.New("player already voted on this poll")
	ErrRegistrationClosed         = errors.New("presentation not found or already started")
	ErrNameTaken                  = errors.New("player name already taken")
	ErrPresentationAlreadyStarted = errors.New("presentation has already started")
	ErrPresentationNotRunning     = errors.New("presentation is not running")
	ErrInvalidPlayerName          = errors.New("player name cannot be empty")
	// ErrForbidden means the logged-in user does not own the presentation.
	ErrForbidden     = errors.New("only the owner of the presentation can do this")
	ErrInvalidOption = errors.New("option key does not exist on this poll")
	// ErrVotingClosed means the poll was revealed or the presentation finished.
	ErrVotingClosed = errors.New("voting on this poll is closed")
	// ErrTimeUp means the poll's countdown ran out.
	ErrTimeUp = errors.New("time is up for this poll")
	// ErrPollAlreadyWon means another player answered correctly first.
	ErrPollAlreadyWon = errors.New("another player already answered this poll correctly")
	// ErrInvalidPlayer means the player id and token do not match.
	ErrInvalidPlayer = errors.New("player id or token is invalid")
	// ErrInvalidRefreshToken covers unknown, expired and revoked refresh tokens.
	ErrInvalidRefreshToken = errors.New("refresh token is invalid or expired")
)

// ValidationError is returned when input is well-formed JSON but breaks a
// rule (empty question, too many options, ...). Its message is safe to show.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }
