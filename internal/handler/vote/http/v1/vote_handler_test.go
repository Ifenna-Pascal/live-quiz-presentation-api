package v1

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/sirupsen/logrus"
)

type fakeVoteUsecase struct{ err error }

func (f fakeVoteUsecase) CreateVote(context.Context, uuid.UUID, model.CreateVoteRequestDTO) (model.VoteResult, error) {
	return model.VoteResult{Scored: true, Correct: true, Won: true}, f.err
}
func (f fakeVoteUsecase) GetPollVotes(context.Context, uuid.UUID, uuid.UUID) ([]db.GetVotesRow, error) {
	return []db.GetVotesRow{}, f.err
}
func (f fakeVoteUsecase) ListPresentationVotes(context.Context, uuid.UUID, uuid.UUID) ([]model.PresentationVoteResponse, error) {
	return nil, f.err
}
func (f fakeVoteUsecase) GetLeaderboard(context.Context, uuid.UUID) (model.LeaderboardResponse, error) {
	return model.LeaderboardResponse{}, f.err
}

const validVote = `{"poll_id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","key":"A","player_id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","player_token":"t"}`

func TestCreateVoteHandlerStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	id := uuid.NewString()

	cases := []struct {
		name string
		path string
		body string
		err  error
		want int
	}{
		{"recorded", id, validVote, nil, 200},
		{"bad presentation id", "nope", validVote, nil, 400},
		{"missing token", id, `{"poll_id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","key":"A","player_id":"7c9e6679-7425-40de-944b-e07fc1f90ae7"}`, nil, 400},
		{"not json", id, `nope`, nil, 400},
		{"not the poll on screen", id, validVote, model.ErrIDMismatch, 400},
		{"unknown option", id, validVote, model.ErrInvalidOption, 400},
		{"wrong player token", id, validVote, model.ErrPlayerNotRegistered, 403},
		{"presentation missing", id, validVote, model.ErrNotFound, 404},
		{"already voted", id, validVote, model.ErrAlreadyVoted, 409},
		{"voting closed", id, validVote, model.ErrVotingClosed, 409},
		{"time is up", id, validVote, model.ErrTimeUp, 409},
		{"someone won first", id, validVote, model.ErrPollAlreadyWon, 409},
	}
	for _, tc := range cases {
		router := gin.New()
		router.POST("/presentations/:presentation_id/votes", NewVoteHandler(logger, fakeVoteUsecase{tc.err}).CreateVoteHandler)
		req := httptest.NewRequest(http.MethodPost, "/presentations/"+tc.path+"/votes", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d (%s)", tc.name, w.Code, tc.want, w.Body.String())
		}
		if tc.want == 200 && !strings.Contains(w.Body.String(), `"won":true`) {
			t.Errorf("the answer result must be returned, got %s", w.Body.String())
		}
	}
}
