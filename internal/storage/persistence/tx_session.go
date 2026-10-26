package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

// RotateSessionTx exchanges a refresh token for a new one: the old session is
// revoked and a new session is created for the same user.
//
// If the old token was already revoked, someone is replaying it (it may have
// been stolen), so every session of that user is revoked.
func (store *SQLStore) RotateSessionTx(ctx context.Context, arg RotateSessionParams) (Session, error) {
	var created Session
	reused := false
	err := store.execTx(ctx, func(q *Queries) error {
		old, err := q.GetSessionByTokenHash(ctx, arg.OldTokenHash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrInvalidRefreshToken
			}
			return err
		}
		if old.Isrevoked {
			reused = true
			return q.RevokeAllUserSessions(ctx, old.Userid)
		}
		if time.Now().After(old.Expiresat) {
			return model.ErrInvalidRefreshToken
		}
		if err := q.RevokeSession(ctx, old.ID); err != nil {
			return err
		}
		created, err = q.CreateSession(ctx, CreateSessionParams{
			UserID:           old.Userid,
			RefreshTokenHash: arg.NewTokenHash,
			UserAgent:        arg.UserAgent,
			ClientIp:         arg.ClientIP,
			ExpiresAt:        arg.ExpiresAt,
		})
		return err
	})
	if err != nil {
		return Session{}, err
	}
	if reused {
		// Committed the revocation above; report the token as invalid.
		return Session{}, model.ErrInvalidRefreshToken
	}
	return created, nil
}
