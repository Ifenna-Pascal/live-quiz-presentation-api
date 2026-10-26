package model

import "time"

// PresentionResponseDTO is a presentation with all its polls. Correct answers
// are included for the owner, and for polls that have been revealed.
type PresentionResponseDTO struct {
	ID               string                     `json:"id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	Title            string                     `json:"title" example:"Friday tech quiz"`
	CurrentPollIndex int32                      `json:"current_poll_index" example:"1"`
	HasStarted       bool                       `json:"has_started" example:"false"`
	HasFinished      bool                       `json:"has_finished" example:"false"`
	IsOwner          bool                       `json:"is_owner" example:"false"`
	// Prize is null when the presentation has no prize money.
	Prize *Prize                     `json:"prize"`
	Polls []PresentationResponsePoll `json:"polls"`
}

// Prize is the money the top 3 ranks share, in whole units of Currency.
type Prize struct {
	Pool     int64  `json:"pool" example:"50000"`
	Currency string `json:"currency" example:"NGN"`
}

type PresentationResponsePoll struct {
	ID               string   `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Question         string   `json:"question"`
	TimeLimitSeconds int32    `json:"time_limit_seconds" example:"20"`
	Revealed         bool     `json:"revealed"`
	Options          []Option `json:"options"`
}

// CreateUserResponseDTO represents the response model for a created user.
// It never contains the password hash.
type CreateUserResponseDTO struct {
	ID    string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Name  string `json:"name" example:"Ada Lovelace"`
	Email string `json:"email" example:"ada@example.com"`
}

// LoginUserResponseDTO is returned by login and by refresh. The access token
// is short lived; the refresh token is single use and gets a new pair.
type LoginUserResponseDTO struct {
	ID                    string    `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Name                  string    `json:"name" example:"Ada Lovelace"`
	Email                 string    `json:"email" example:"ada@example.com"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// VoteResponse represents the response model for a vote.
type VoteResponse struct {
	Key        string `json:"key" example:"A"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}

// Reasons a poll stopped taking answers.
const (
	ClosedByWinner   = "won"      // the first correct answer won it
	ClosedByTimeUp   = "time_up"  // the countdown ran out
	ClosedByHost     = "revealed" // the host revealed the answer early
	ClosedByFinished = "finished" // the presentation finished
)

// CurrentPoll is the poll on screen. Options carry the correct flag only once
// the poll is closed (Revealed).
type CurrentPoll struct {
	ID    string `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Index int32  `json:"index" example:"1"`
	// Revealed is true once the poll no longer takes answers; the correct
	// options are then included.
	Revealed bool `json:"revealed" example:"false"`
	// Scored is true when the poll has a correct option (a quiz question).
	// Opinion polls take one answer per player; scored polls allow retries.
	Scored bool `json:"scored" example:"true"`
	// ClosedReason says why it closed: won, time_up, revealed or finished.
	ClosedReason string `json:"closed_reason,omitempty" example:"won" enums:"won,time_up,revealed,finished"`
	// OpenedAt and ClosesAt bound the countdown. Compare ClosesAt with
	// ServerTime (not the device clock) to show the time left.
	OpenedAt   *time.Time  `json:"opened_at"`
	ClosesAt   *time.Time  `json:"closes_at"`
	ServerTime time.Time   `json:"server_time"`
	Winner     *PollWinner `json:"winner"`
	Poll
}

// PollWinner is the player who answered the poll correctly first.
type PollWinner struct {
	PlayerID   string    `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string    `json:"player_name" example:"Ada"`
	Key        string    `json:"key" example:"B"`
	ElapsedMs  int64     `json:"elapsed_ms" example:"3200"`
	WonAt      time.Time `json:"won_at"`
}

// VoteResult tells the player how their answer went. Scored is false for
// opinion polls (no correct option), where every answer is simply recorded.
type VoteResult struct {
	Correct   bool  `json:"correct"`
	Won       bool  `json:"won"`
	Scored    bool  `json:"scored"`
	ElapsedMs int64 `json:"elapsed_ms,omitempty" example:"3200"`
}

// CreatePresentationResponse represents the response model for a created presentation.
type CreatePresentationResponse struct {
	PresentationID string `json:"presentation_id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
}

// ErrorResponse represents the body returned when a request fails.
type ErrorResponse struct {
	Error string `json:"error" example:"The presentation ran out of polls."`
}

// CreatedPresentationPlayerResponse is a player as everyone sees them.
type CreatedPresentationPlayerResponse struct {
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}

// RegisteredPlayerResponse is returned once, at registration. PlayerToken is
// the secret the player must send to vote and chat; it cannot be recovered.
type RegisteredPlayerResponse struct {
	PlayerID    string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName  string `json:"player_name" example:"Ada"`
	PlayerToken string `json:"player_token" example:"q3Yx..."`
}

// MyPresentationResponse is one row of "my presentations".
type MyPresentationResponse struct {
	ID               string    `json:"id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	Title            string    `json:"title" example:"Friday tech quiz"`
	CurrentPollIndex int32     `json:"current_poll_index" example:"1"`
	HasStarted       bool      `json:"has_started" example:"false"`
	HasFinished      bool      `json:"has_finished" example:"false"`
	PollCount        int32     `json:"poll_count" example:"5"`
	PlayerCount      int32     `json:"player_count" example:"12"`
	Prize            *Prize    `json:"prize"`
	CreatedAt        time.Time `json:"created_at" example:"2026-10-23T14:00:00Z"`
}

type LeaderboardEntry struct {
	Rank       int32  `json:"rank" example:"1"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
	// Score is the number of revealed polls answered correctly.
	Score int32 `json:"score" example:"3"`
	// Answered is the number of polls the player voted on.
	Answered int32 `json:"answered" example:"4"`
	// Payout is this player's current share of the prize (0 when there is
	// no prize, or the player is outside the top 3 ranks). It changes live
	// as scores change, and is final once the presentation finishes.
	Payout int64 `json:"payout" example:"25000"`
}

type LeaderboardResponse struct {
	// ScoredPolls is the highest score possible so far.
	ScoredPolls int32 `json:"scored_polls" example:"4"`
	// Prize is null when the presentation has no prize money.
	Prize   *Prize             `json:"prize"`
	Entries []LeaderboardEntry `json:"entries"`
}

type PresentationVoteResponse struct {
	PollID     string `json:"poll_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PollIndex  int32  `json:"poll_index" example:"1"`
	Key        string `json:"key" example:"A"`
	IsCorrect  bool   `json:"is_correct" example:"true"`
	PlayerID   string `json:"player_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	PlayerName string `json:"player_name" example:"Ada"`
}

// MessageResponse is one chat message.
type MessageResponse struct {
	ID         string    `json:"id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	AuthorID   string    `json:"author_id" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	AuthorName string    `json:"author_name" example:"Ada"`
	AuthorRole string    `json:"author_role" example:"player" enums:"host,player"`
	Body       string    `json:"body" example:"Good luck everyone!"`
	CreatedAt  time.Time `json:"created_at" example:"2026-10-23T14:00:00.123456Z"`
}
