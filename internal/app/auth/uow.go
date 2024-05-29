package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/userstorage"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/user"
)

type UserStorage interface {
	Add(ctx context.Context, u *user.User) error
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByID(ctx context.Context, userId string) (*user.User, error)
	Persist(ctx context.Context, u *user.User) error
	CollectEvents() []domain.Event
	Close() error
}

type AtomicContext struct {
	storage.DBContext
	UserStorage
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

func NewAtomicContext(dbContext storage.DBContext) (*AtomicContext, error) {
	return &AtomicContext{
		DBContext:   dbContext,
		UserStorage: userstorage.NewPostgresStorage(dbContext, nil),
	}, nil
}
