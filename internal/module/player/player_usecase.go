package player

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/lib/pq"
)

// uniqueViolation is the Postgres error code for a broken UNIQUE constraint.
const uniqueViolation = "23505"

// Usecase contains the function of business logic of domain player
type Usecase interface {
	RegisterPlayer(ctx context.Context, presentationID uuid.UUID, param model.RegisterPlayerToPresentation) (model.CreatedPresentationPlayerResponse, error)
	ListPlayers(ctx context.Context, presentationID uuid.UUID) ([]model.CreatedPresentationPlayerResponse, error)
	StartPresentation(ctx context.Context, userID, presentationID uuid.UUID) error
}

type service struct {
	store db.Store
}

// Initialize takes all necessary service for domain player to run the business logic of domain player
func Initialize(store db.Store) Usecase {
	return &service{store: store}
}

func (s service) RegisterPlayer(ctx context.Context, presentationID uuid.UUID, param model.RegisterPlayerToPresentation) (model.CreatedPresentationPlayerResponse, error) {
	name := strings.TrimSpace(param.PlayerName)
	if name == "" {
		return model.CreatedPresentationPlayerResponse{}, model.ErrInvalidPlayerName
	}

	player, err := s.store.AddPlayerToPresentation(ctx, db.AddPlayerToPresentationParams{
		PlayerName:     name,
		PresentationID: presentationID,
	})
	if err != nil {
		// No row inserted: the presentation does not exist or has already started.
		if errors.Is(err, sql.ErrNoRows) {
			return model.CreatedPresentationPlayerResponse{}, model.ErrRegistrationClosed
		}
		// UNIQUE(presentationID, playerName): the name is already registered.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
			return model.CreatedPresentationPlayerResponse{}, model.ErrNameTaken
		}
		return model.CreatedPresentationPlayerResponse{}, err
	}

	return model.CreatedPresentationPlayerResponse{
		PlayerID:   player.ID.String(),
		PlayerName: player.Playername,
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
	// Only the owner may start the presentation.
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return err
	}

	// The presentation exists, so no row updated means it had already started.
	if _, err := s.store.StartPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrPresentationAlreadyStarted
		}
		return err
	}
	return nil
}
