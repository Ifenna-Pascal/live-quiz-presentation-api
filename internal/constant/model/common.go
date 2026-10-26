package model

type Poll struct {
	Question string `json:"question" binding:"required" example:"What is your favourite language?"`
	// TimeLimitSeconds is how long players have to answer (5-600). 0 means
	// the default of 20 seconds.
	TimeLimitSeconds int32    `json:"time_limit_seconds" example:"20"`
	Options          []Option `json:"options" binding:"required,dive"`
}

type Option struct {
	Key   string `json:"key" db:"optionKey" binding:"required" example:"A"`
	Value string `json:"value" db:"optionValue" binding:"required" example:"Go"`
	// Correct marks the right answer. Public responses only include it once
	// the poll is revealed; the owner always sees it. omitempty keeps false
	// (and hidden) values out of the JSON.
	Correct bool `json:"correct,omitempty" example:"true"`
}

// HideAnswers returns a copy of options without the correct flags.
func HideAnswers(options []Option) []Option {
	hidden := make([]Option, len(options))
	for i, o := range options {
		hidden[i] = Option{Key: o.Key, Value: o.Value}
	}
	return hidden
}

// NewPrize returns nil when there is no prize money.
func NewPrize(pool int64, valid bool, currency string) *Prize {
	if !valid || pool <= 0 {
		return nil
	}
	return &Prize{Pool: pool, Currency: currency}
}
