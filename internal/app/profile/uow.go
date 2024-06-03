package profileapp

import (
	"context"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	profilestorage "github.com/burenotti/go_health_backend/internal/adapter/storage/profiles"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
)

type AtomicContext struct {
	ctx            context.Context
	dbContext      storage.DBContext
	ProfileStorage ProfileStorage
}

type ProfileStorage interface {
	Add(ctx context.Context, profile profile.Profile) error
	GetByID(ctx context.Context, userId string) (profile.Profile, error)
	CollectEvents() []domain.Event
	Close() error
}

func NewAtomicContext(
	ctx context.Context,
	dbContext storage.DBContext,
) (*AtomicContext, error) {
	return &AtomicContext{
		ctx:            ctx,
		dbContext:      dbContext,
		ProfileStorage: profilestorage.NewPostgresStorage(dbContext),
	}, nil
}

func (a *AtomicContext) Context() context.Context {
	return a.ctx
}

func (a *AtomicContext) Commit() error {
	return a.dbContext.Commit()
}

func (a *AtomicContext) Close() error {
	return a.ProfileStorage.Close()
}

func (a *AtomicContext) CollectEvents() []domain.Event {
	return a.ProfileStorage.CollectEvents()
}
