package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

// Store is every query plus the operations that need a transaction.
type Store interface {
	Querier
	GetCurrentPoll(context.Context, uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPollToForwardTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPollToPreviousTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error)
	VoteCurrentPollTx(context.Context, VoteParams) (model.VoteResult, error)
	StartPresentationTx(ctx context.Context, presentationID uuid.UUID) error
	GetOwnedPresentation(ctx context.Context, userID, presentationID uuid.UUID) (Presentation, error)
	CreatePresentationTx(ctx context.Context, userID uuid.UUID, details PresentationDetails, polls []byte) (uuid.UUID, error)
	ReplacePresentationTx(ctx context.Context, presentationID uuid.UUID, details PresentationDetails, polls []byte) error
	RotateSessionTx(ctx context.Context, arg RotateSessionParams) (Session, error)
	Ping(ctx context.Context) error
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

// Ping checks that the database answers.
func (store *SQLStore) Ping(ctx context.Context) error {
	return store.db.PingContext(ctx)
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
