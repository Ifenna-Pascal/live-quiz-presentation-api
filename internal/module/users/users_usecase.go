package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
	"github.com/ifenna-pascal/interactive-presentation/pkg/util"
)

// uniqueViolation is the Postgres error code for a broken UNIQUE constraint.
const uniqueViolation = "23505"

type Usecase interface {
	CreateUser(ctx context.Context, user *model.CreateUserRequestDTO) (model.CreateUserResponseDTO, error)
	LoginUser(ctx context.Context, user *model.LoginUserRequestDTO) (model.LoginUserResponseDTO, error)
}

type service struct {
	store         db.Store
	tokenMaker    token.Maker
	tokenDuration time.Duration
}

func Initialize(
	store db.Store,
	tokenMaker token.Maker,
	tokenDuration time.Duration,
) Usecase {
	return &service{
		store:         store,
		tokenMaker:    tokenMaker,
		tokenDuration: tokenDuration,
	}
}

func (s service) CreateUser(ctx context.Context, user *model.CreateUserRequestDTO) (model.CreateUserResponseDTO, error) {
	hashedPassword, err := util.HashPassword(user.Password)
	if err != nil {
		return model.CreateUserResponseDTO{}, err
	}

	created, err := s.store.CreateUser(ctx, db.CreateUserParams{
		Name:           user.Name,
		Email:          user.Email,
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

func (s service) LoginUser(ctx context.Context, userParam *model.LoginUserRequestDTO) (model.LoginUserResponseDTO, error) {
	user, err := s.store.GetUser(ctx, userParam.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.LoginUserResponseDTO{}, model.ErrInvalidCredentials
		}
		return model.LoginUserResponseDTO{}, err
	}

	if err := util.CheckPassword(userParam.Password, user.Hashedpassword); err != nil {
		return model.LoginUserResponseDTO{}, model.ErrInvalidCredentials
	}

	accessToken, err := s.tokenMaker.CreateToken(user.ID.String(), s.tokenDuration)
	if err != nil {
		return model.LoginUserResponseDTO{}, err
	}

	return model.LoginUserResponseDTO{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
		Token: accessToken,
	}, nil
}
