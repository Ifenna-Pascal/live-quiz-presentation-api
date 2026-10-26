package respond

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/sirupsen/logrus"
)

func init() { gin.SetMode(gin.TestMode) }

func run(err error) (int, model.ErrorResponse) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	Error(c, logger, err)
	var body model.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// Every domain error maps to one status code and a {"error": ...} body.
func TestErrorStatusCodes(t *testing.T) {
	cases := map[error]int{
		model.ErrNotFound:                   404,
		model.ErrIDMismatch:                 400,
		model.ErrNoPollDisplayed:            400,
		model.ErrRunOutOfIndex:              400,
		model.ErrEmailTaken:                 409,
		model.ErrInvalidCredentials:         401,
		model.ErrPlayerNotRegistered:        403,
		model.ErrAlreadyVoted:               409,
		model.ErrRegistrationClosed:         409,
		model.ErrNameTaken:                  409,
		model.ErrPresentationAlreadyStarted: 409,
		model.ErrPresentationNotRunning:     409,
		model.ErrInvalidPlayerName:          400,
		model.ErrForbidden:                  403,
		model.ErrInvalidOption:              400,
		model.ErrVotingClosed:               409,
		model.ErrInvalidPlayer:              403,
		model.ErrInvalidRefreshToken:        401,
		model.ValidationError{Message: "x"}: 400,
		errors.New("database exploded"):     500,
	}
	for err, want := range cases {
		got, body := run(err)
		if got != want {
			t.Errorf("%v: got %d, want %d", err, got, want)
		}
		if body.Error == "" {
			t.Errorf("%v: empty error message", err)
		}
	}
}

func TestWrappedErrorsAreRecognised(t *testing.T) {
	got, _ := run(fmt.Errorf("context: %w", model.ErrForbidden))
	if got != 403 {
		t.Fatalf("got %d, want 403", got)
	}
}

func TestInternalErrorsDoNotLeak(t *testing.T) {
	_, body := run(errors.New("pq: password authentication failed"))
	if body.Error != "Something went wrong. Please try again." {
		t.Fatalf("leaked: %q", body.Error)
	}
}

func TestValidationMessageIsShown(t *testing.T) {
	_, body := run(model.ValidationError{Message: "Poll 1 needs a question."})
	if body.Error != "Poll 1 needs a question." {
		t.Fatalf("got %q", body.Error)
	}
}
