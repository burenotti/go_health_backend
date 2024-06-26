package authapp

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/userstorage"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
)

type UserStorage interface {
	Add(ctx context.Context, u *auth.User) error
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	GetByID(ctx context.Context, userId string) (*auth.User, error)
	GetByAuthID(ctx context.Context, authId string) (*auth.User, error)
	GetByAuthSecret(ctx context.Context, authId string) (*auth.User, error)
	Persist(ctx context.Context, u *auth.User) error
	CollectEvents() []domain.Event
	Close() error
}

type AtomicContext struct {
	ctx context.Context
	storage.DBContext
	UserStorage UserStorage
}

func (a *AtomicContext) Commit() error {
	return a.DBContext.Commit()
}

func (a *AtomicContext) Close() (err error) {
	if closeErr := a.UserStorage.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	if err != nil {
		err = errors.Join(fmt.Errorf("failed to close storage"), err)
	}

	return err
}

func (a *AtomicContext) CollectEvents() []domain.Event {
	return a.UserStorage.CollectEvents()
}

func (a *AtomicContext) Context() context.Context {
	return a.ctx
}

func NewAtomicContext(ctx context.Context, dbContext storage.DBContext) (*AtomicContext, error) {
	return &AtomicContext{
		ctx:         ctx,
		DBContext:   dbContext,
		UserStorage: userstorage.NewPostgresStorage(dbContext, nil),
	}, nil
}
