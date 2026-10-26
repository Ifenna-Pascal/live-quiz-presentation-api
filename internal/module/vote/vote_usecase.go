package vote

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/util"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
	"github.com/lib/pq"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

type Usecase interface {
	CreateVote(ctx context.Context, presentationID uuid.UUID, param model.CreateVoteRequestDTO) (model.VoteResult, error)
	GetPollVotes(ctx context.Context, presentationID, pollID uuid.UUID) ([]db.GetVotesRow, error)
	ListPresentationVotes(ctx context.Context, userID, presentationID uuid.UUID) ([]model.PresentationVoteResponse, error)
	GetLeaderboard(ctx context.Context, presentationID uuid.UUID) (model.LeaderboardResponse, error)
}

type service struct {
	store     db.Store
	publisher events.Publisher
}

func Initialize(store db.Store, publisher events.Publisher) Usecase {
	return &service{store: store, publisher: publisher}
}

func (s service) CreateVote(ctx context.Context, presentationID uuid.UUID, param model.CreateVoteRequestDTO) (model.VoteResult, error) {
	result, err := s.store.VoteCurrentPollTx(ctx, db.VoteParams{
		PresentationID:   presentationID,
		Pollid:           param.PollID,
		Optionkey:        param.Key,
		PlayerID:         param.PlayerID,
		PlayerSecretHash: util.HashSecret(param.PlayerToken),
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case uniqueViolation:
				return model.VoteResult{}, model.ErrAlreadyVoted
			case foreignKeyViolation:
				return model.VoteResult{}, model.ErrInvalidOption
			}
		}
		return model.VoteResult{}, err
	}
	s.publisher.Publish(ctx, presentationID, events.TypeVote, map[string]string{"poll_id": param.PollID.String()})
	if result.Won {
		s.announceWinner(ctx, presentationID)
	}
	return result, nil
}

// announceWinner tells every screen who answered first, then that the poll closed.
func (s service) announceWinner(ctx context.Context, presentationID uuid.UUID) {
	if poll, err := s.store.GetCurrentPoll(ctx, presentationID); err == nil && poll.Winner != nil {
		s.publisher.Publish(ctx, presentationID, events.TypeWinner, struct {
			PollID string `json:"poll_id"`
			*model.PollWinner
		}{poll.ID, poll.Winner})
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
}

func (s service) GetPollVotes(ctx context.Context, presentationID, pollID uuid.UUID) ([]db.GetVotesRow, error) {
	if _, err := s.store.GetPresentationAndPoll(ctx, db.GetPresentationAndPollParams{
		PresentationID: presentationID,
		PollID:         pollID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []db.GetVotesRow{}, model.ErrNotFound
		}
		return []db.GetVotesRow{}, err
	}
	return s.store.GetVotes(ctx, db.GetVotesParams{
		PresentationID: presentationID,
		PollID:         pollID,
	})
}

func (s service) ListPresentationVotes(ctx context.Context, userID, presentationID uuid.UUID) ([]model.PresentationVoteResponse, error) {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return []model.PresentationVoteResponse{}, err
	}

	rows, err := s.store.ListPresentationVotes(ctx, presentationID)
	if err != nil {
		return []model.PresentationVoteResponse{}, err
	}

	votes := make([]model.PresentationVoteResponse, 0, len(rows))
	for _, row := range rows {
		votes = append(votes, model.PresentationVoteResponse{
			PollID:     row.PollID.String(),
			PollIndex:  row.PollIndex,
			Key:        row.Key,
			IsCorrect:  row.IsCorrect,
			PlayerID:   row.PlayerID.String(),
			PlayerName: row.PlayerName,
		})
	}
	return votes, nil
}

// GetLeaderboard is public: only revealed polls count, so it never leaks
// an answer before the host shows it.
func (s service) GetLeaderboard(ctx context.Context, presentationID uuid.UUID) (model.LeaderboardResponse, error) {
	pres, err := s.store.GetPresentation(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.LeaderboardResponse{}, model.ErrNotFound
		}
		return model.LeaderboardResponse{}, err
	}

	scoredPolls, err := s.store.CountScoredPolls(ctx, presentationID)
	if err != nil {
		return model.LeaderboardResponse{}, err
	}
	rows, err := s.store.GetLeaderboard(ctx, presentationID)
	if err != nil {
		return model.LeaderboardResponse{}, err
	}

	entries := make([]model.LeaderboardEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, model.LeaderboardEntry{
			Rank:       row.Rank,
			PlayerID:   row.PlayerID.String(),
			PlayerName: row.PlayerName,
			Score:      row.Score,
			Answered:   row.Answered,
		})
	}
	prize := model.NewPrize(pres.Prizepool.Int64, pres.Prizepool.Valid, pres.Prizecurrency)
	if prize != nil {
		SplitPrize(prize.Pool, entries)
	}
	return model.LeaderboardResponse{ScoredPolls: scoredPolls, Prize: prize, Entries: entries}, nil
}
