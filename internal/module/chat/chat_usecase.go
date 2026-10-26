package chat

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/util"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
)

const (
	maxBodyLength = 500
	// DefaultLimit and MaxLimit bound how many messages one request returns.
	DefaultLimit = 100
	MaxLimit     = 200
)

type Usecase interface {
	// SendMessage posts as the player in req (id + token), or else as the
	// logged-in owner (userID).
	SendMessage(ctx context.Context, presentationID uuid.UUID, userID *uuid.UUID, req model.CreateMessageRequestDTO) (model.MessageResponse, error)
	// ListMessages returns messages oldest first: the newest `limit` ones, or
	// the ones after `after`.
	ListMessages(ctx context.Context, presentationID uuid.UUID, after *time.Time, limit int32) ([]model.MessageResponse, error)
}

type service struct {
	store     db.Store
	publisher events.Publisher
}

func Initialize(store db.Store, publisher events.Publisher) Usecase {
	return &service{store: store, publisher: publisher}
}

func (s service) SendMessage(ctx context.Context, presentationID uuid.UUID, userID *uuid.UUID, req model.CreateMessageRequestDTO) (model.MessageResponse, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return model.MessageResponse{}, model.ValidationError{Message: "The message cannot be empty."}
	}
	if utf8.RuneCountInString(body) > maxBodyLength {
		return model.MessageResponse{}, model.ValidationError{Message: "A message can be at most 500 characters."}
	}

	pres, err := s.store.GetPresentation(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.MessageResponse{}, model.ErrNotFound
		}
		return model.MessageResponse{}, err
	}

	params := db.CreateMessageParams{PresentationID: presentationID, Body: body}
	switch {
	case req.PlayerID != nil:
		player, err := s.store.GetPlayerInPresentation(ctx, db.GetPlayerInPresentationParams{
			PlayerID:       *req.PlayerID,
			PresentationID: presentationID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.MessageResponse{}, model.ErrInvalidPlayer
			}
			return model.MessageResponse{}, err
		}
		if !player.Secrethash.Valid || !util.SecretMatchesHash(req.PlayerToken, player.Secrethash.String) {
			return model.MessageResponse{}, model.ErrInvalidPlayer
		}
		params.PlayerID = uuid.NullUUID{UUID: player.ID, Valid: true}
		params.AuthorID = player.ID
		params.AuthorName = player.Playername
		params.AuthorRole = "player"
	case userID != nil:
		if !pres.Userid.Valid || pres.Userid.UUID != *userID {
			return model.MessageResponse{}, model.ErrForbidden
		}
		user, err := s.store.GetUserByID(ctx, *userID)
		if err != nil {
			return model.MessageResponse{}, err
		}
		params.AuthorID = user.ID
		params.AuthorName = user.Name
		params.AuthorRole = "host"
	default:
		return model.MessageResponse{}, model.ErrInvalidPlayer
	}

	msg, err := s.store.CreateMessage(ctx, params)
	if err != nil {
		return model.MessageResponse{}, err
	}
	resp := toResponse(msg)
	s.publisher.Publish(ctx, presentationID, events.TypeChat, resp)
	return resp, nil
}

func (s service) ListMessages(ctx context.Context, presentationID uuid.UUID, after *time.Time, limit int32) ([]model.MessageResponse, error) {
	if _, err := s.store.GetPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []model.MessageResponse{}, model.ErrNotFound
		}
		return []model.MessageResponse{}, err
	}

	var rows []db.Message
	var err error
	if after != nil {
		rows, err = s.store.ListMessagesAfter(ctx, db.ListMessagesAfterParams{PresentationID: presentationID, After: *after, PageLimit: limit})
	} else {
		rows, err = s.store.ListRecentMessages(ctx, db.ListRecentMessagesParams{PresentationID: presentationID, PageLimit: limit})
	}
	if err != nil {
		return []model.MessageResponse{}, err
	}

	messages := make([]model.MessageResponse, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, toResponse(row))
	}
	return messages, nil
}

func toResponse(m db.Message) model.MessageResponse {
	return model.MessageResponse{
		ID:         m.ID.String(),
		AuthorID:   m.Authorid.String(),
		AuthorName: m.Authorname,
		AuthorRole: m.Authorrole,
		Body:       m.Body,
		CreatedAt:  m.Createdat,
	}
}
