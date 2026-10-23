package vote

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/lib/pq"
)

// uniqueViolation is the Postgres error code for a broken UNIQUE constraint.
const uniqueViolation = "23505"

// foreignKeyViolation is the Postgres error code for a broken FOREIGN KEY constraint.
const foreignKeyViolation = "23503"

// Usecase contains the function of business logic of domain voter
type Usecase interface {
	CreateVote(ctx context.Context, presentationID uuid.UUID, param model.CreateVoteRequestDTO) error
	GetPollVotes(ctx context.Context, presentationID, pollID uuid.UUID) ([]db.GetVotesRow, error)
	ListPresentationVotes(ctx context.Context, userID, presentationID uuid.UUID) ([]model.PresentationVoteResponse, error)
	GetLeaderboard(ctx context.Context, userID, presentationID uuid.UUID) (model.LeaderboardResponse, error)
}

type service struct {
	store db.Store
}

// Initialize takes all necessary service for domain vote to run the business logic of domain vote
func Initialize(
	store db.Store,
) Usecase {
	return &service{store: store}
}

func (s service) CreateVote(ctx context.Context, presentationID uuid.UUID, param model.CreateVoteRequestDTO) error {
	err := s.store.VoteCurrentPollTx(context.Background(), db.VoteParams{
		PresentationID: presentationID,
		Pollid:         param.PollID,
		Optionkey:      param.Key,
		PlayerID:       param.PlayerID,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			// UNIQUE(pollID, playerID): the player already voted on this poll.
			case uniqueViolation:
				return model.ErrAlreadyVoted
			// FOREIGN KEY (pollID, optionKey): the key is not an option of this poll.
			case foreignKeyViolation:
				return model.ErrInvalidOption
			}
		}
		return err
	}
	return nil
}

func (s service) GetPollVotes(ctx context.Context, presentationID, pollID uuid.UUID) ([]db.GetVotesRow, error) {
	// Any poll of the presentation can be read, not only the current one.
	if _, err := s.store.GetPresentationAndPoll(context.Background(), db.GetPresentationAndPollParams{
		PresentationID: presentationID,
		PollID:         pollID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []db.GetVotesRow{}, model.ErrNotFound
		}
		return []db.GetVotesRow{}, err
	}
	votes, err := s.store.GetVotes(context.Background(), db.GetVotesParams{
		PresentationID: presentationID,
		PollID:         pollID,
	})
	return votes, err
}

// ListPresentationVotes returns every vote of every poll. Only the owner may
// read it, because it shows which votes were correct.
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

// GetLeaderboard ranks the players by the number of polls they answered
// correctly. Only the owner may read it: if players could, someone who has not
// voted yet could watch the scores change and work out the right answer.
func (s service) GetLeaderboard(ctx context.Context, userID, presentationID uuid.UUID) (model.LeaderboardResponse, error) {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
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
	return model.LeaderboardResponse{ScoredPolls: scoredPolls, Entries: entries}, nil
}
