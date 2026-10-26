package presentation

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
)

const (
	maxTitleLength    = 120
	maxPolls          = 50
	maxQuestionLength = 300
	minOptions        = 2
	maxOptions        = 6
	maxKeyLength      = 8
	maxValueLength    = 200
	defaultTimeLimit  = 20
	minTimeLimit      = 5
	maxTimeLimit      = 600
	maxPrizePool      = 1_000_000_000_000
	defaultCurrency   = "NGN"
)

// currencies are the prize currencies the app can display.
var currencies = map[string]bool{"NGN": true, "USD": true, "GBP": true, "EUR": true, "GHS": true, "KES": true, "ZAR": true}

// normalize trims every text field and checks the presentation's rules.
// It returns a model.ValidationError describing the first problem found.
func normalize(req *model.CreatePresentionRequestDTO) error {
	req.Title = strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(req.Title) > maxTitleLength {
		return invalid("The title can be at most %d characters.", maxTitleLength)
	}
	if err := normalizePrize(req); err != nil {
		return err
	}
	if len(req.Polls) == 0 {
		return invalid("Add at least one poll.")
	}
	if len(req.Polls) > maxPolls {
		return invalid("A presentation can have at most %d polls.", maxPolls)
	}

	for i := range req.Polls {
		poll := &req.Polls[i]
		n := i + 1
		poll.Question = strings.TrimSpace(poll.Question)
		if poll.Question == "" {
			return invalid("Poll %d needs a question.", n)
		}
		if utf8.RuneCountInString(poll.Question) > maxQuestionLength {
			return invalid("Poll %d: the question can be at most %d characters.", n, maxQuestionLength)
		}
		if poll.TimeLimitSeconds == 0 {
			poll.TimeLimitSeconds = defaultTimeLimit
		}
		if poll.TimeLimitSeconds < minTimeLimit || poll.TimeLimitSeconds > maxTimeLimit {
			return invalid("Poll %d: the time limit must be between %d and %d seconds.", n, minTimeLimit, maxTimeLimit)
		}
		if len(poll.Options) < minOptions || len(poll.Options) > maxOptions {
			return invalid("Poll %d needs between %d and %d options.", n, minOptions, maxOptions)
		}
		seen := map[string]bool{}
		for j := range poll.Options {
			opt := &poll.Options[j]
			opt.Key = strings.TrimSpace(opt.Key)
			opt.Value = strings.TrimSpace(opt.Value)
			if opt.Key == "" || utf8.RuneCountInString(opt.Key) > maxKeyLength {
				return invalid("Poll %d: every option needs a key of 1 to %d characters.", n, maxKeyLength)
			}
			if seen[opt.Key] {
				return invalid("Poll %d: option key %q is used twice.", n, opt.Key)
			}
			seen[opt.Key] = true
			if opt.Value == "" {
				return invalid("Poll %d: option %s needs an answer.", n, opt.Key)
			}
			if utf8.RuneCountInString(opt.Value) > maxValueLength {
				return invalid("Poll %d: option %s can be at most %d characters.", n, opt.Key, maxValueLength)
			}
		}
	}
	return nil
}

// normalizePrize treats a missing or zero prize as "no prize money".
func normalizePrize(req *model.CreatePresentionRequestDTO) error {
	req.PrizeCurrency = strings.ToUpper(strings.TrimSpace(req.PrizeCurrency))
	if req.PrizeCurrency == "" {
		req.PrizeCurrency = defaultCurrency
	}
	if !currencies[req.PrizeCurrency] {
		return invalid("The prize currency %q is not supported.", req.PrizeCurrency)
	}
	if req.PrizePool == nil || *req.PrizePool == 0 {
		req.PrizePool = nil
		return nil
	}
	if *req.PrizePool < 0 || *req.PrizePool > maxPrizePool {
		return invalid("The prize must be between 1 and %d.", int64(maxPrizePool))
	}
	return nil
}

// details is the presentation's title and prize, ready to store.
func details(req *model.CreatePresentionRequestDTO) db.PresentationDetails {
	d := db.PresentationDetails{Title: req.Title, PrizeCurrency: req.PrizeCurrency}
	if req.PrizePool != nil {
		d.PrizePool = sql.NullInt64{Int64: *req.PrizePool, Valid: true}
	}
	return d
}

func invalid(format string, args ...any) error {
	return model.ValidationError{Message: fmt.Sprintf(format, args...)}
}
