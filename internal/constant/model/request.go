package model

import "github.com/google/uuid"

// CreatePresentionRequestDTO is the body for creating or replacing a
// presentation. The owner is the logged-in user, taken from the access token.
type CreatePresentionRequestDTO struct {
	Title string `json:"title" example:"Friday tech quiz"`
	// PrizePool is optional prize money, in whole units of PrizeCurrency.
	// Leave it out (or send 0) for a game without money. The top 3 ranks
	// share it in proportion to their scores.
	PrizePool *int64 `json:"prize_pool,omitempty" example:"50000"`
	// PrizeCurrency is an ISO 4217 code. Defaults to NGN.
	PrizeCurrency string `json:"prize_currency,omitempty" example:"NGN"`
	Polls         []Poll `json:"polls"`
}

// CreateVoteRequestDTO represents the request model for creating a vote.
// PlayerToken is the secret returned when the player registered.
type CreateVoteRequestDTO struct {
	PollID      uuid.UUID `json:"poll_id" binding:"required" swaggertype:"string" format:"uuid" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Key         string    `json:"key" binding:"required" example:"A"`
	PlayerID    uuid.UUID `json:"player_id" binding:"required" swaggertype:"string" format:"uuid" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerToken string    `json:"player_token" binding:"required" example:"q3Yx..."`
}

// CreateUserRequestDTO represents the request model for creating a user.
// The password is sent in plain text and hashed before it is stored.
type CreateUserRequestDTO struct {
	Name     string `json:"name" binding:"required,max=80" example:"Ada Lovelace"`
	Email    string `json:"email" binding:"required,email,max=254" example:"ada@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"s3cretpass"`
}

type LoginUserRequestDTO struct {
	Email    string `json:"email" binding:"required,email" example:"ada@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"s3cretpass"`
}

// RefreshTokenRequestDTO is the body for refreshing and for logging out.
type RefreshTokenRequestDTO struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"bX9k..."`
}

// RegisterPlayerToPresentation represents the request model for registering a player.
// The presentation is taken from the URL.
type RegisterPlayerToPresentation struct {
	PlayerName string `json:"player_name" binding:"required" example:"Ada"`
}

// CreateMessageRequestDTO is a chat message. Players send their id and
// token; the host sends neither and is identified by the access token.
type CreateMessageRequestDTO struct {
	Body        string     `json:"body" binding:"required" example:"Good luck everyone!"`
	PlayerID    *uuid.UUID `json:"player_id" swaggertype:"string" format:"uuid"`
	PlayerToken string     `json:"player_token"`
}
