package model

import "github.com/google/uuid"

// CreatePresentationRequest represents the request model for creating a presentation.
// The owner is the logged-in user, taken from the access token.
type CreatePresentionRequestDTO struct {
	Polls []Poll `json:"polls"`
}

// CreateVoteRequest represents the request model for creating a vote.
type CreateVoteRequestDTO struct {
	PollID   uuid.UUID `json:"poll_id" binding:"required" swaggertype:"string" format:"uuid" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Key      string    `json:"key" binding:"required" example:"A"`
	PlayerID uuid.UUID `json:"player_id" binding:"required" swaggertype:"string" format:"uuid" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
}

// CreateUserRequestDTO represents the request model for creating a user.
// The password is sent in plain text and hashed before it is stored.
type CreateUserRequestDTO struct {
	Name     string `json:"name" binding:"required" example:"Ada Lovelace"`
	Email    string `json:"email" binding:"required,email" example:"ada@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"s3cretpass"`
}

type LoginUserRequestDTO struct {
	Email    string `json:"email" binding:"required,email" example:"ada@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"s3cretpass"`
}

// RegisterPlayerToPresentation represents the request model for registering a player.
// The presentation is taken from the URL.
type RegisterPlayerToPresentation struct {
	PlayerName string `json:"player_name" binding:"required" example:"Ada"`
}
