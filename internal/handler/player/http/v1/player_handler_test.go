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
	"github.com/sirupsen/logrus"
)

type fakePlayerUsecase struct{ err error }

func (f fakePlayerUsecase) RegisterPlayer(context.Context, uuid.UUID, model.RegisterPlayerToPresentation) (model.RegisteredPlayerResponse, error) {
	return model.RegisteredPlayerResponse{PlayerID: "p", PlayerName: "Ada", PlayerToken: "secret"}, f.err
}
func (f fakePlayerUsecase) ListPlayers(context.Context, uuid.UUID) ([]model.CreatedPresentationPlayerResponse, error) {
	return nil, f.err
}
func (f fakePlayerUsecase) StartPresentation(context.Context, uuid.UUID, uuid.UUID) error {
	return f.err
}
func (f fakePlayerUsecase) RemovePlayer(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return f.err
}

func TestRegisterPlayerHandlerStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	id := uuid.NewString()

	cases := []struct {
		name string
		body string
		err  error
		want int
	}{
		{"joined", `{"player_name":"Ada"}`, nil, 201},
		{"missing name", `{}`, nil, 400},
		{"empty name", `{"player_name":"Ada"}`, model.ErrInvalidPlayerName, 400},
		{"too long", `{"player_name":"Ada"}`, model.ValidationError{Message: "too long"}, 400},
		{"name taken", `{"player_name":"Ada"}`, model.ErrNameTaken, 409},
		{"registration closed", `{"player_name":"Ada"}`, model.ErrRegistrationClosed, 409},
	}
	for _, tc := range cases {
		router := gin.New()
		router.POST("/presentations/:presentation_id/players", NewPlayerHandler(logger, fakePlayerUsecase{tc.err}).RegisterPlayerHandler)
		req := httptest.NewRequest(http.MethodPost, "/presentations/"+id+"/players", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, w.Code, tc.want)
		}
		if tc.want == 201 && !strings.Contains(w.Body.String(), `"player_token":"secret"`) {
			t.Errorf("registration must return the player token, got %s", w.Body.String())
		}
	}
}

func TestOwnerActionsStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	id := uuid.NewString()

	for err, want := range map[error]int{nil: 204, model.ErrForbidden: 403, model.ErrNotFound: 404, model.ErrPresentationAlreadyStarted: 409} {
		router := gin.New()
		router.POST("/presentations/:presentation_id/start", NewPlayerHandler(logger, fakePlayerUsecase{err}).StartPresentationHandler)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/presentations/"+id+"/start", nil))
		if w.Code != want {
			t.Errorf("start with %v: got %d, want %d", err, w.Code, want)
		}
	}
}
