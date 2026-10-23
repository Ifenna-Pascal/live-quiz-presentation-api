package model

import (
	"encoding/json"
	"time"
)

// PresentionResponseDTO represents the request model for creating a presentation.
type PresentionResponseDTO struct {
	ID               string `json:"id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CurrentPollIndex int32  `json:"current_poll_index" example:"1"`
	// HasStarted is false while players can still register.
	HasStarted bool                       `json:"has_started" example:"false"`
	Polls      []PresentationResponsePoll `json:"polls"`
}

// CreateUserResponseDTO represents the response model for a created user.
// It never contains the password hash.
type CreateUserResponseDTO struct {
	ID    string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Name  string `json:"name" example:"Ada Lovelace"`
	Email string `json:"email" example:"ada@example.com"`
}

type LoginUserResponseDTO struct {
	ID    string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Name  string `json:"name" example:"Ada Lovelace"`
	Email string `json:"email" example:"ada@example.com"`
	Token string `json:"token"`
}

// VoteResponse represents the response model for a vote.
type VoteResponse struct {
	Key        string `json:"key" example:"A"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}

type PresentationResponsePoll struct {
	Question string          `json:"question" binding:"required"`
	Options  json.RawMessage `json:"options" binding:"required,dive" swaggertype:"array,object"`
}

type CurrentPoll struct {
	ID string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Poll
}

// CreatePresentationResponse represents the response model for a created presentation.
type CreatePresentationResponse struct {
	PresentationID string `json:"presentation_id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
}

// ErrorResponse represents the body returned when a request fails.
type ErrorResponse struct {
	Error string `json:"error" example:"The presentation ran out of polls."`
}

type CreatedPresentationPlayerResponse struct {
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}

// MyPresentationResponse is one entry of the logged-in user's presentations.
type MyPresentationResponse struct {
	ID               string    `json:"id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CurrentPollIndex int32     `json:"current_poll_index" example:"1"`
	HasStarted       bool      `json:"has_started" example:"false"`
	PollCount        int32     `json:"poll_count" example:"5"`
	CreatedAt        time.Time `json:"created_at" example:"2026-10-23T14:00:00Z"`
}

// LeaderboardEntry is the score of one player.
type LeaderboardEntry struct {
	// Rank starts at 1. Players with the same score share a rank.
	Rank       int32  `json:"rank" example:"1"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
	// Score is the number of polls answered correctly.
	Score int32 `json:"score" example:"3"`
	// Answered is the number of polls the player voted on.
	Answered int32 `json:"answered" example:"4"`
}

// LeaderboardResponse ranks every registered player of a presentation.
type LeaderboardResponse struct {
	// ScoredPolls is the highest score possible so far: displayed polls that
	// have a correct answer.
	ScoredPolls int32              `json:"scored_polls" example:"4"`
	Entries     []LeaderboardEntry `json:"entries"`
}

// PresentationVoteResponse is one vote of any poll of a presentation.
type PresentationVoteResponse struct {
	PollID     string `json:"poll_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PollIndex  int32  `json:"poll_index" example:"1"`
	Key        string `json:"key" example:"A"`
	IsCorrect  bool   `json:"is_correct" example:"true"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}
