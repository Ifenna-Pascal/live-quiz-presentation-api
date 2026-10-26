package persistence

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// VoteParams is the input of SQLStore.VoteCurrentPollTx (see tx_vote.go).
type VoteParams struct {
	// The presentation, taken from the URL. Used only for the "is this the
	// current poll?" check; it is not a column of the votes table.
	PresentationID uuid.UUID
	Pollid         uuid.UUID `db:"pollid"`
	Optionkey      string    `db:"optionkey"`
	PlayerID       uuid.UUID `db:"playerid"`
	// PlayerSecretHash is the hash of the token the player sent.
	PlayerSecretHash string
}

// RotateSessionParams is the input of SQLStore.RotateSessionTx.
type RotateSessionParams struct {
	OldTokenHash string
	NewTokenHash string
	UserAgent    string
	ClientIP     string
	ExpiresAt    time.Time
}

// PresentationDetails is everything about a presentation except its polls.
// PrizePool is NULL when there is no prize money.
type PresentationDetails struct {
	Title         string
	PrizePool     sql.NullInt64
	PrizeCurrency string
}
