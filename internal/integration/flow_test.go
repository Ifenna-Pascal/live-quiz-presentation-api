// Package integration runs the use cases against a real Postgres database.
//
// It is skipped unless TEST_DB_SOURCE is set. The database it points to is
// wiped, so never point it at real data:
//
//	docker exec interactive-presentation-postgres-1 createdb -U root ip_test
//	TEST_DB_SOURCE="postgresql://root:secret@localhost:5432/ip_test?sslmode=disable" go test ./internal/integration/
package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/chat"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/player"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/presentation"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/users"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/vote"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
)

func openTestStore(t *testing.T) (db.Store, *sql.DB) {
	t.Helper()
	source := os.Getenv("TEST_DB_SOURCE")
	if source == "" {
		t.Skip("set TEST_DB_SOURCE to run integration tests")
	}
	conn, err := sql.Open("postgres", source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	m, err := migrate.New("file://../../db/migration", source)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}
	return db.NewStore(conn), conn
}

func mustErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func TestFullGameFlow(t *testing.T) {
	store, conn := openTestStore(t)
	ctx := context.Background()
	pub := events.NopPublisher{}
	maker, _ := token.NewJWTMaker("integration-test-secret-key-0123456789")

	userUC := users.Initialize(store, maker, time.Minute, time.Hour)
	presUC := presentation.Initialize(store, pub)
	playerUC := player.Initialize(store, pub)
	voteUC := vote.Initialize(store, pub)
	chatUC := chat.Initialize(store, pub)

	// --- accounts and refresh-token rotation ---
	if _, err := userUC.CreateUser(ctx, &model.CreateUserRequestDTO{Name: "Host", Email: "host@example.com", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	_, err := userUC.CreateUser(ctx, &model.CreateUserRequestDTO{Name: "Host", Email: "HOST@example.com", Password: "password1"})
	mustErr(t, err, model.ErrEmailTaken)

	client := users.ClientInfo{UserAgent: "test", IP: "127.0.0.1"}
	login, err := userUC.LoginUser(ctx, &model.LoginUserRequestDTO{Email: "host@example.com", Password: "password1"}, client)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := userUC.RefreshSession(ctx, login.RefreshToken, client)
	if err != nil || refreshed.RefreshToken == login.RefreshToken {
		t.Fatalf("refresh should rotate the token: %v", err)
	}
	_, err = userUC.RefreshSession(ctx, login.RefreshToken, client)
	mustErr(t, err, model.ErrInvalidRefreshToken)
	_, err = userUC.RefreshSession(ctx, refreshed.RefreshToken, client)
	mustErr(t, err, model.ErrInvalidRefreshToken) // reuse revoked every session
	hostID := uuid.MustParse(login.ID)

	// --- create, edit ---
	req := model.CreatePresentionRequestDTO{Title: "Quiz", Polls: []model.Poll{
		{Question: "Red planet?", Options: []model.Option{{Key: "A", Value: "Venus"}, {Key: "B", Value: "Mars", Correct: true}}},
		{Question: "Opinion?", TimeLimitSeconds: 30, Options: []model.Option{{Key: "A", Value: "Yes"}, {Key: "B", Value: "No"}}},
	}}
	presID, err := presUC.CreatePresentation(ctx, hostID, &req)
	if err != nil {
		t.Fatal(err)
	}
	req.Title = "Space quiz"
	if err := presUC.UpdatePresentation(ctx, hostID, presID, &req); err != nil {
		t.Fatal(err)
	}
	mustErr(t, presUC.UpdatePresentation(ctx, uuid.New(), presID, &req), model.ErrForbidden)

	public, _ := presUC.GetPresentation(ctx, nil, presID)
	if public.Title != "Space quiz" || public.Polls[0].Options[1].Correct {
		t.Fatalf("public view must hide answers: %+v", public)
	}
	owner, _ := presUC.GetPresentation(ctx, &hostID, presID)
	if !owner.IsOwner || !owner.Polls[0].Options[1].Correct {
		t.Fatalf("owner view must show answers: %+v", owner)
	}

	// --- players ---
	ada, err := playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "  Ada  "})
	if err != nil || ada.PlayerToken == "" || ada.PlayerName != "Ada" {
		t.Fatalf("register: %+v %v", ada, err)
	}
	_, err = playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "Ada"})
	mustErr(t, err, model.ErrNameTaken)
	bo, _ := playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "Bo"})

	if err := playerUC.StartPresentation(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}
	mustErr(t, presUC.UpdatePresentation(ctx, hostID, presID, &req), model.ErrPresentationAlreadyStarted)
	_, err = playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "Late"})
	mustErr(t, err, model.ErrRegistrationClosed)

	// --- timed voting: wrong answers can retry, the first correct one wins ---
	poll, _ := presUC.GetCurrentPoll(ctx, presID)
	if poll.OpenedAt == nil || poll.ClosesAt == nil || poll.TimeLimitSeconds != 20 || poll.Revealed {
		t.Fatalf("poll 1 should be open with a 20s countdown: %+v", poll)
	}
	adaID, boID := uuid.MustParse(ada.PlayerID), uuid.MustParse(bo.PlayerID)
	pollID := uuid.MustParse(poll.ID)
	vote := func(player uuid.UUID, token, key string) (model.VoteResult, error) {
		return voteUC.CreateVote(ctx, presID, model.CreateVoteRequestDTO{PollID: pollID, Key: key, PlayerID: player, PlayerToken: token})
	}

	_, err = vote(adaID, bo.PlayerToken, "B")
	mustErr(t, err, model.ErrPlayerNotRegistered)
	res, err := vote(adaID, ada.PlayerToken, "A")
	if err != nil || res.Correct || res.Won || !res.Scored {
		t.Fatalf("A is wrong: %+v %v", res, err)
	}
	_, err = vote(adaID, ada.PlayerToken, "A")
	mustErr(t, err, model.ErrAlreadyVoted) // same option twice
	_, err = vote(boID, bo.PlayerToken, "Z")
	mustErr(t, err, model.ErrInvalidOption)

	board, _ := voteUC.GetLeaderboard(ctx, presID)
	if board.Entries[0].Score != 0 {
		t.Fatalf("nobody has won yet, got %+v", board)
	}

	res, err = vote(boID, bo.PlayerToken, "B")
	if err != nil || !res.Won || !res.Correct {
		t.Fatalf("Bo should win: %+v %v", res, err)
	}
	_, err = vote(adaID, ada.PlayerToken, "B")
	mustErr(t, err, model.ErrPollAlreadyWon) // too late, even with the right answer

	poll, _ = presUC.GetCurrentPoll(ctx, presID)
	if !poll.Revealed || poll.ClosedReason != model.ClosedByWinner || poll.Winner == nil || poll.Winner.PlayerName != "Bo" || poll.Winner.Key != "B" {
		t.Fatalf("poll should be closed and won by Bo: %+v", poll)
	}
	if !poll.Options[1].Correct {
		t.Fatal("a closed poll shows its correct answer")
	}
	board, _ = voteUC.GetLeaderboard(ctx, presID)
	if board.ScoredPolls != 1 || board.Entries[0].PlayerName != "Bo" || board.Entries[0].Score != 1 || board.Entries[0].Rank != 1 {
		t.Fatalf("Bo should lead with 1 point, got %+v", board)
	}

	// --- opinion poll: one answer each, then the time runs out ---
	if _, err := presUC.UpdateCurrentPoll(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}
	poll, _ = presUC.GetCurrentPoll(ctx, presID)
	pollID = uuid.MustParse(poll.ID)
	if poll.TimeLimitSeconds != 30 || poll.OpenedAt == nil {
		t.Fatalf("poll 2 should be open with 30s: %+v", poll)
	}
	res, err = vote(adaID, ada.PlayerToken, "A")
	if err != nil || res.Scored || res.Won {
		t.Fatalf("opinion vote: %+v %v", res, err)
	}
	_, err = vote(adaID, ada.PlayerToken, "B")
	mustErr(t, err, model.ErrAlreadyVoted)

	if _, err := conn.Exec(`UPDATE polls SET openedAt = now() - interval '1 minute' WHERE id = $1`, pollID); err != nil {
		t.Fatal(err)
	}
	_, err = vote(boID, bo.PlayerToken, "A")
	mustErr(t, err, model.ErrTimeUp)
	poll, _ = presUC.GetCurrentPoll(ctx, presID)
	if poll.ClosedReason != model.ClosedByTimeUp || !poll.Revealed {
		t.Fatalf("poll 2 should be timed out: %+v", poll)
	}
	if p, err := presUC.PreviousPoll(ctx, hostID, presID); err != nil || p.Index != 1 || p.ClosedReason != model.ClosedByWinner {
		t.Fatalf("going back must not reopen poll 1: %+v %v", p, err)
	}
	if _, err := presUC.UpdateCurrentPoll(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}

	// --- chat ---
	if _, err := chatUC.SendMessage(ctx, presID, nil, model.CreateMessageRequestDTO{Body: "gg", PlayerID: &adaID, PlayerToken: ada.PlayerToken}); err != nil {
		t.Fatal(err)
	}
	_, err = chatUC.SendMessage(ctx, presID, nil, model.CreateMessageRequestDTO{Body: "gg", PlayerID: &adaID, PlayerToken: "wrong"})
	mustErr(t, err, model.ErrInvalidPlayer)
	if _, err := chatUC.SendMessage(ctx, presID, &hostID, model.CreateMessageRequestDTO{Body: "Welcome"}); err != nil {
		t.Fatal(err)
	}
	stranger := uuid.New()
	_, err = chatUC.SendMessage(ctx, presID, &stranger, model.CreateMessageRequestDTO{Body: "hi"})
	mustErr(t, err, model.ErrForbidden)
	msgs, _ := chatUC.ListMessages(ctx, presID, nil, 10)
	if len(msgs) != 2 || msgs[0].AuthorRole != "player" || msgs[1].AuthorRole != "host" {
		t.Fatalf("messages: %+v", msgs)
	}
	after, _ := chatUC.ListMessages(ctx, presID, &msgs[0].CreatedAt, 10)
	if len(after) != 1 || after[0].Body != "Welcome" {
		t.Fatalf("after: %+v", after)
	}

	// --- navigation, removal, finish, delete ---
	_, err = presUC.UpdateCurrentPoll(ctx, hostID, presID)
	mustErr(t, err, model.ErrRunOutOfIndex)

	if err := playerUC.RemovePlayer(ctx, hostID, presID, boID); err != nil {
		t.Fatal(err)
	}
	mustErr(t, playerUC.RemovePlayer(ctx, hostID, presID, boID), model.ErrNotFound)

	if err := presUC.FinishPresentation(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}
	mustErr(t, presUC.FinishPresentation(ctx, hostID, presID), model.ErrPresentationNotRunning)

	if err := presUC.DeletePresentation(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}
	_, err = presUC.GetPresentation(ctx, nil, presID)
	mustErr(t, err, model.ErrNotFound)
}

// Twenty players hit the right answer at the same moment: exactly one wins,
// everyone else is told someone was first, and only one point is given.
func TestOnlyOneWinnerWhenAnswersRace(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	pub := events.NopPublisher{}
	maker, _ := token.NewJWTMaker("integration-test-secret-key-0123456789")
	userUC := users.Initialize(store, maker, time.Minute, time.Hour)
	presUC := presentation.Initialize(store, pub)
	playerUC := player.Initialize(store, pub)
	voteUC := vote.Initialize(store, pub)

	user, _ := userUC.CreateUser(ctx, &model.CreateUserRequestDTO{Name: "Host", Email: "race@example.com", Password: "password1"})
	hostID := uuid.MustParse(user.ID)
	presID, err := presUC.CreatePresentation(ctx, hostID, &model.CreatePresentionRequestDTO{Polls: []model.Poll{
		{Question: "2+2?", Options: []model.Option{{Key: "A", Value: "4", Correct: true}, {Key: "B", Value: "5"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	const n = 20
	players := make([]model.RegisteredPlayerResponse, n)
	for i := range players {
		players[i], _ = playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "P" + strconv.Itoa(i)})
	}
	if err := playerUC.StartPresentation(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}
	poll, _ := presUC.GetCurrentPoll(ctx, presID)

	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, lost := 0, 0
	start := make(chan struct{})
	for _, p := range players {
		wg.Add(1)
		go func(p model.RegisteredPlayerResponse) {
			defer wg.Done()
			<-start
			res, err := voteUC.CreateVote(ctx, presID, model.CreateVoteRequestDTO{
				PollID: uuid.MustParse(poll.ID), Key: "A", PlayerID: uuid.MustParse(p.PlayerID), PlayerToken: p.PlayerToken,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && res.Won:
				wins++
			case errors.Is(err, model.ErrPollAlreadyWon):
				lost++
			default:
				t.Errorf("unexpected result %+v %v", res, err)
			}
		}(p)
	}
	close(start)
	wg.Wait()

	if wins != 1 || lost != n-1 {
		t.Fatalf("want 1 winner and %d too late, got %d and %d", n-1, wins, lost)
	}
	board, _ := voteUC.GetLeaderboard(ctx, presID)
	total := 0
	for _, e := range board.Entries {
		total += int(e.Score)
	}
	if total != 1 {
		t.Fatalf("exactly one point should be given, got %d", total)
	}
}

func TestPrizeIsSharedLive(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	pub := events.NopPublisher{}
	maker, _ := token.NewJWTMaker("integration-test-secret-key-0123456789")
	userUC := users.Initialize(store, maker, time.Minute, time.Hour)
	presUC := presentation.Initialize(store, pub)
	playerUC := player.Initialize(store, pub)
	voteUC := vote.Initialize(store, pub)

	if _, err := userUC.CreateUser(ctx, &model.CreateUserRequestDTO{Name: "Host", Email: "prize@example.com", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	login, _ := userUC.LoginUser(ctx, &model.LoginUserRequestDTO{Email: "prize@example.com", Password: "password1"}, users.ClientInfo{})
	hostID := uuid.MustParse(login.ID)

	q := model.Poll{Question: "2+2?", Options: []model.Option{{Key: "A", Value: "3"}, {Key: "B", Value: "4", Correct: true}}}
	pool := int64(50000)

	// Without a prize, nothing is paid and the prize is null.
	plainID, err := presUC.CreatePresentation(ctx, hostID, &model.CreatePresentionRequestDTO{Polls: []model.Poll{q}})
	if err != nil {
		t.Fatal(err)
	}
	if plain, _ := presUC.GetPresentation(ctx, nil, plainID); plain.Prize != nil {
		t.Fatalf("no prize expected, got %+v", plain.Prize)
	}

	_, err = presUC.CreatePresentation(ctx, hostID, &model.CreatePresentionRequestDTO{PrizePool: &pool, PrizeCurrency: "XYZ", Polls: []model.Poll{q}})
	var verr model.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("unknown currency should fail validation, got %v", err)
	}

	presID, err := presUC.CreatePresentation(ctx, hostID, &model.CreatePresentionRequestDTO{PrizePool: &pool, Polls: []model.Poll{q, q, q, q}})
	if err != nil {
		t.Fatal(err)
	}
	pres, _ := presUC.GetPresentation(ctx, nil, presID)
	if pres.Prize == nil || pres.Prize.Pool != 50000 || pres.Prize.Currency != "NGN" {
		t.Fatalf("prize should be 50000 NGN, got %+v", pres.Prize)
	}

	a, _ := playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "A"})
	b, _ := playerUC.RegisterPlayer(ctx, presID, model.RegisterPlayerToPresentation{PlayerName: "B"})
	if err := playerUC.StartPresentation(ctx, hostID, presID); err != nil {
		t.Fatal(err)
	}

	win := func(p model.RegisteredPlayerResponse) {
		t.Helper()
		poll, _ := presUC.GetCurrentPoll(ctx, presID)
		if _, err := voteUC.CreateVote(ctx, presID, model.CreateVoteRequestDTO{
			PollID: uuid.MustParse(poll.ID), Key: "B", PlayerID: uuid.MustParse(p.PlayerID), PlayerToken: p.PlayerToken,
		}); err != nil {
			t.Fatal(err)
		}
		presUC.UpdateCurrentPoll(ctx, hostID, presID)
	}
	payoutOf := func(name string) int64 {
		board, _ := voteUC.GetLeaderboard(ctx, presID)
		for _, e := range board.Entries {
			if e.PlayerName == name {
				return e.Payout
			}
		}
		return -1
	}

	win(a)
	win(a)
	win(a)
	if got := payoutOf("A"); got != 50000 {
		t.Fatalf("a lone scorer takes the whole prize, got %d", got)
	}
	win(b)
	if a, b := payoutOf("A"), payoutOf("B"); a != 37500 || b != 12500 {
		t.Fatalf("3:1 should split 37500/12500, got %d/%d", a, b)
	}

	// The prize is locked once the game starts.
	err = presUC.UpdatePresentation(ctx, hostID, presID, &model.CreatePresentionRequestDTO{Polls: []model.Poll{q}})
	mustErr(t, err, model.ErrPresentationAlreadyStarted)
}
