// Package events delivers real-time updates about a presentation.
//
// Writers call Publisher.Publish after a change is saved. The publisher sends
// a Postgres NOTIFY, every API instance LISTENs, and each instance's Hub
// passes the event to the browsers streaming that presentation (SSE).
package events

import "encoding/json"

// Event types. Most events only say "this changed, fetch it again"; chat
// carries the full message so it appears without another request.
const (
	TypePresentation = "presentation" // started, poll changed, revealed, finished, edited
	TypePlayers      = "players"      // a player joined or was removed
	TypeVote         = "vote"         // data: {"poll_id": "..."}
	TypeChat         = "chat"         // data: model.MessageResponse
	TypeWinner       = "winner"       // data: model.PollWinner + poll_id; someone answered first
	TypeDeleted      = "deleted"      // the presentation was deleted
	// TypeResync asks clients to refetch everything, e.g. after this instance
	// lost its database connection and may have missed events.
	TypeResync = "resync"
)

type Event struct {
	PresentationID string          `json:"presentation_id"`
	Type           string          `json:"type"`
	Data           json.RawMessage `json:"data,omitempty"`
}
