package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

type Store interface {
	Querier
	GetCurrentPoll(context.Context, uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPollToForwardTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPollToPreviousTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error)
	VoteCurrentPollTx(context.Context, VoteParams) error
	GetOwnedPresentation(ctx context.Context, userID, presentationID uuid.UUID) (Presentation, error)
}

type SQLStore struct {
	db *sql.DB
	mu sync.Mutex
	*Queries
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}
