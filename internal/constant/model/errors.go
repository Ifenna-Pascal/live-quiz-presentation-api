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
	ErrInvalidPlayerName          = errors.New("player name cannot be empty")
	// ErrForbidden means the logged-in user does not own the presentation.
	ErrForbidden     = errors.New("only the owner of the presentation can do this")
	ErrInvalidOption = errors.New("option key does not exist on this poll")
)
