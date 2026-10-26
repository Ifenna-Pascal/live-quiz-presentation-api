package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
	"github.com/ifenna-pascal/interactive-presentation/pkg/util"
)

const uniqueViolation = "23505"

// ClientInfo is stored with a session so a user can tell their devices apart.
type ClientInfo struct {
	UserAgent string
	IP        string
}

type Usecase interface {
	CreateUser(ctx context.Context, user *model.CreateUserRequestDTO) (model.CreateUserResponseDTO, error)
	LoginUser(ctx context.Context, user *model.LoginUserRequestDTO, client ClientInfo) (model.LoginUserResponseDTO, error)
	// RefreshSession swaps a refresh token for a new access + refresh token pair.
	RefreshSession(ctx context.Context, refreshToken string, client ClientInfo) (model.LoginUserResponseDTO, error)
	// Logout revokes a refresh token. Unknown tokens are ignored.
	Logout(ctx context.Context, refreshToken string) error
}

type service struct {
	store           db.Store
	tokenMaker      token.Maker
	accessDuration  time.Duration
	refreshDuration time.Duration
	now             func() time.Time
}

func Initialize(
	store db.Store,
	tokenMaker token.Maker,
	accessDuration time.Duration,
	refreshDuration time.Duration,
) Usecase {
	return &service{
		store:           store,
		tokenMaker:      tokenMaker,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
		now:             time.Now,
	}
}

func (s service) CreateUser(ctx context.Context, user *model.CreateUserRequestDTO) (model.CreateUserResponseDTO, error) {
	name := strings.TrimSpace(user.Name)
	if name == "" {
		return model.CreateUserResponseDTO{}, model.ValidationError{Message: "The `name` cannot be empty."}
	}
	hashedPassword, err := util.HashPassword(user.Password)
	if err != nil {
		return model.CreateUserResponseDTO{}, err
	}

	created, err := s.store.CreateUser(ctx, db.CreateUserParams{
		Name:           name,
		Email:          strings.ToLower(strings.TrimSpace(user.Email)),
		Hashedpassword: hashedPassword,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
			return model.CreateUserResponseDTO{}, model.ErrEmailTaken
		}
		return model.CreateUserResponseDTO{}, err
	}

	return model.CreateUserResponseDTO{
		ID:    created.ID.String(),
		Name:  created.Name,
		Email: created.Email,
	}, nil
}

func (s service) LoginUser(ctx context.Context, userParam *model.LoginUserRequestDTO, client ClientInfo) (model.LoginUserResponseDTO, error) {
	user, err := s.store.GetUser(ctx, strings.ToLower(strings.TrimSpace(userParam.Email)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.LoginUserResponseDTO{}, model.ErrInvalidCredentials
		}
		return model.LoginUserResponseDTO{}, err
	}
	if err := util.CheckPassword(userParam.Password, user.Hashedpassword); err != nil {
		return model.LoginUserResponseDTO{}, model.ErrInvalidCredentials
	}

	refreshToken, refreshHash, err := util.NewSecret()
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}
	session, err := s.store.CreateSession(ctx, db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		UserAgent:        truncate(client.UserAgent, 255),
		ClientIp:         client.IP,
		ExpiresAt:        s.now().Add(s.refreshDuration),
	})
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}
	return s.tokenPair(user, refreshToken, session.Expiresat)
}

func (s service) RefreshSession(ctx context.Context, refreshToken string, client ClientInfo) (model.LoginUserResponseDTO, error) {
	newToken, newHash, err := util.NewSecret()
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}
	session, err := s.store.RotateSessionTx(ctx, db.RotateSessionParams{
		OldTokenHash: util.HashSecret(refreshToken),
		NewTokenHash: newHash,
		UserAgent:    truncate(client.UserAgent, 255),
		ClientIP:     client.IP,
		ExpiresAt:    s.now().Add(s.refreshDuration),
	})
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}
	user, err := s.store.GetUserByID(ctx, session.Userid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.LoginUserResponseDTO{}, model.ErrInvalidRefreshToken
		}
		return model.LoginUserResponseDTO{}, err
	}
	return s.tokenPair(user, newToken, session.Expiresat)
}

func (s service) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.store.GetSessionByTokenHash(ctx, util.HashSecret(refreshToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return s.store.RevokeSession(ctx, session.ID)
}

func (s service) tokenPair(user db.User, refreshToken string, refreshExpiresAt time.Time) (model.LoginUserResponseDTO, error) {
	accessExpiresAt := s.now().Add(s.accessDuration)
	accessToken, err := s.tokenMaker.CreateToken(user.ID.String(), s.accessDuration)
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}
	return model.LoginUserResponseDTO{
		ID:                    user.ID.String(),
		Name:                  user.Name,
		Email:                 user.Email,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
