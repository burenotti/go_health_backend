package groupservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/groups"
	profilestorage "github.com/burenotti/go_health_backend/internal/adapter/storage/profiles"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/group"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
)

type GroupStorage interface {
	Add(ctx context.Context, g *group.Group) error
	GetByID(ctx context.Context, groupID group.GroupID) (*group.Group, error)
	GetMembers(ctx context.Context, groupID group.GroupID, limit, offset int) ([]*group.Member, error)

	ListByTrainee(
		ctx context.Context,
		traineeID group.TraineeID,
		limit, offset int,
	) (map[group.GroupID]*group.Group, error)

	ListByCoach(
		ctx context.Context,
		coachID group.CoachID,
		limit, offset int,
	) (map[group.GroupID]*group.Group, error)

	Close() error
	CollectEvents() []domain.Event
}

type ProfilesStorage interface {
	GetByID(ctx context.Context, profileID string) (profile.Profile, error)
	Close() error
	CollectEvents() []domain.Event
}

type AtomicContext struct {
	ctx context.Context
	storage.DBContext
	GroupStorage    GroupStorage
	ProfilesStorage ProfilesStorage
}

func (a *AtomicContext) Context() context.Context {
	return a.ctx
}

func (a *AtomicContext) Commit() error {
	return a.DBContext.Commit()
}

func (a *AtomicContext) Close() (err error) {
	if closeErr := a.GroupStorage.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	if closeErr := a.ProfilesStorage.Close(); closeErr != nil {
		errors.Join(err, closeErr)
	}

	if err != nil {
		err = errors.Join(fmt.Errorf("failed to close storage"), err)
	}

	return err
}

func (a *AtomicContext) CollectEvents() []domain.Event {
	groupEvents := a.GroupStorage.CollectEvents()
	profileEvents := a.ProfilesStorage.CollectEvents()

	events := make([]domain.Event, 0, len(groupEvents)+len(profileEvents))
	events = append(events, groupEvents...)
	events = append(events, profileEvents...)
	return events
}

func NewAtomicContext(ctx context.Context, dbContext storage.DBContext) (*AtomicContext, error) {
	return &AtomicContext{
		ctx:             ctx,
		DBContext:       dbContext,
		GroupStorage:    groupstorage.NewPostgresStorage(dbContext, nil),
		ProfilesStorage: profilestorage.NewPostgresStorage(dbContext),
	}, nil
}
