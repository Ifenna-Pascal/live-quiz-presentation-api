package model

import (
	"github.com/google/uuid"
)

type Poll struct {
	Question string   `json:"question" binding:"required" example:"What is your favourite language?"`
	Options  []Option `json:"options" binding:"required,dive"`
}

type Option struct {
	Key   string `json:"key" db:"optionKey" binding:"required" example:"A"`
	Value string `json:"value" db:"optionValue" binding:"required" example:"Go"`
	// Correct marks the right answer when creating a presentation. It is never
	// filled in responses (the poll queries do not select it), so players
	// cannot see the answer; omitempty keeps it out of the JSON.
	Correct bool `json:"correct,omitempty" example:"true"`
}

type Vote struct {
	PollID         uuid.UUID
	PresentationID uuid.UUID
	Options        []string
	VoteCount      int
}
