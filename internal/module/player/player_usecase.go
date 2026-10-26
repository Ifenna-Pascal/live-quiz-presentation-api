package player

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/util"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
	"github.com/lib/pq"
)

const (
	uniqueViolation     = "23505"
	maxPlayerNameLength = 30
)

type Usecase interface {
	RegisterPlayer(ctx context.Context, presentationID uuid.UUID, param model.RegisterPlayerToPresentation) (model.RegisteredPlayerResponse, error)
	ListPlayers(ctx context.Context, presentationID uuid.UUID) ([]model.CreatedPresentationPlayerResponse, error)
	StartPresentation(ctx context.Context, userID, presentationID uuid.UUID) error
	RemovePlayer(ctx context.Context, userID, presentationID, playerID uuid.UUID) error
}

type service struct {
	store     db.Store
	publisher events.Publisher
}

func Initialize(store db.Store, publisher events.Publisher) Usecase {
	return &service{store: store, publisher: publisher}
}

// RegisterPlayer adds a player while registration is open and returns the
// player's secret token. Only its hash is stored.
func (s service) RegisterPlayer(ctx context.Context, presentationID uuid.UUID, param model.RegisterPlayerToPresentation) (model.RegisteredPlayerResponse, error) {
	name := strings.Join(strings.Fields(param.PlayerName), " ")
	if name == "" {
		return model.RegisteredPlayerResponse{}, model.ErrInvalidPlayerName
	}
	if utf8.RuneCountInString(name) > maxPlayerNameLength {
		return model.RegisteredPlayerResponse{}, model.ValidationError{Message: "The `player_name` can be at most 30 characters."}
	}

	secret, hash, err := util.NewSecret()
	if err != nil {
		return model.RegisteredPlayerResponse{}, err
	}

	player, err := s.store.AddPlayerToPresentation(ctx, db.AddPlayerToPresentationParams{
		PlayerName:     name,
		SecretHash:     hash,
		PresentationID: presentationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RegisteredPlayerResponse{}, model.ErrRegistrationClosed
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
			return model.RegisteredPlayerResponse{}, model.ErrNameTaken
		}
		return model.RegisteredPlayerResponse{}, err
	}

	s.publisher.Publish(ctx, presentationID, events.TypePlayers, nil)
	return model.RegisteredPlayerResponse{
		PlayerID:    player.ID.String(),
		PlayerName:  player.Playername,
		PlayerToken: secret,
	}, nil
}

func (s service) ListPlayers(ctx context.Context, presentationID uuid.UUID) ([]model.CreatedPresentationPlayerResponse, error) {
	// Check the presentation first, so an unknown id gives "not found"
	// instead of an empty list.
	if _, err := s.store.GetPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []model.CreatedPresentationPlayerResponse{}, model.ErrNotFound
		}
		return []model.CreatedPresentationPlayerResponse{}, err
	}

	rows, err := s.store.GetPlayersInPresentation(ctx, presentationID)
	if err != nil {
		return []model.CreatedPresentationPlayerResponse{}, err
	}

	players := make([]model.CreatedPresentationPlayerResponse, 0, len(rows))
	for _, row := range rows {
		players = append(players, model.CreatedPresentationPlayerResponse{
			PlayerID:   row.ID.String(),
			PlayerName: row.Name,
		})
	}
	return players, nil
}

func (s service) StartPresentation(ctx context.Context, userID, presentationID uuid.UUID) error {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return err
	}
	if err := s.store.StartPresentationTx(ctx, presentationID); err != nil {
		return err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return nil
}

// RemovePlayer deletes a player and their votes. Only the owner can do it.
func (s service) RemovePlayer(ctx context.Context, userID, presentationID, playerID uuid.UUID) error {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return err
	}
	rows, err := s.store.RemovePlayerFromPresentation(ctx, db.RemovePlayerFromPresentationParams{
		PlayerID:       playerID,
		PresentationID: presentationID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	s.publisher.Publish(ctx, presentationID, events.TypePlayers, nil)
	return nil
}
