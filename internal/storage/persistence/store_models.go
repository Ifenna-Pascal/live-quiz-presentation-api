package persistence

// The UUID type.
import "github.com/google/uuid"

// VoteParams is the input of SQLStore.VoteCurrentPollTx (see tx_vote.go).
type VoteParams struct {
	// The presentation, taken from the URL. Used only for the "is this the
	// current poll?" check; it is not a column of the votes table.
	PresentationID uuid.UUID
	// The three values that are actually inserted into the votes table.
	Pollid    uuid.UUID `db:"pollid"`
	Optionkey string    `db:"optionkey"`
	PlayerID  uuid.UUID `db:"playerid"`
}
