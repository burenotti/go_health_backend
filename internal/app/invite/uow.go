package inviteservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	invitesstorage "github.com/burenotti/go_health_backend/internal/adapter/storage/invites"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/invite"
)

type InvitesStorage interface {
	Add(ctx context.Context, i *invite.Invite) error
	Persist(ctx context.Context, i *invite.Invite) error
	GetByID(ctx context.Context, inviteId invite.InviteID) (*invite.Invite, error)
	GetBySecret(ctx context.Context, secret string) (*invite.Invite, error)

	Close() error
	CollectEvents() []domain.Event
}

type AtomicContext struct {
	ctx            context.Context
	db             storage.DBContext
	InvitesStorage InvitesStorage
}

func (a *AtomicContext) Context() context.Context {
	return a.ctx
}

func (a *AtomicContext) Commit() error {
	return a.db.Commit()
}

func (a *AtomicContext) Close() (err error) {
	if closeErr := a.InvitesStorage.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	if err != nil {
		err = errors.Join(fmt.Errorf("failed to close storage"), err)
	}

	return err
}

func (a *AtomicContext) CollectEvents() []domain.Event {
	return a.InvitesStorage.CollectEvents()
}

func NewAtomicContext(ctx context.Context, dbContext storage.DBContext) (*AtomicContext, error) {
	return &AtomicContext{
		ctx:            ctx,
		db:             dbContext,
		InvitesStorage: invitesstorage.NewPostgresStorage(dbContext, nil),
	}, nil
}
