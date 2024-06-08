package groupservice

import (
	"context"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/group"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
	"github.com/samber/lo"
	"log/slog"
)

type Service struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) CreateGroup(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	groupID group.GroupID,
	coachID group.CoachID,
	name string,
	description string,
) error {
	return uow.Atomic(ctx, func(ctx *AtomicContext) error {
		g := group.New(groupID, coachID, name, description)
		if err := ctx.GroupStorage.Add(ctx.Context(), g); err != nil {
			return err
		}
		return ctx.Commit()
	})
}

func (s *Service) GetByID(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	groupID group.GroupID,
) (g *group.Group, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		var err error
		g, err = ctx.GroupStorage.GetByID(ctx.Context(), groupID)
		return err
	})
	return
}

func (s *Service) GetMembers(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	groupID group.GroupID,
	limit int,
	offset int,
) (m []*group.Member, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		var err error
		m, err = ctx.GroupStorage.GetMembers(ctx.Context(), groupID, limit, offset)
		return err
	})
	return
}

func (s *Service) GetUserGroups(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	userID string,
	limit int,
	offset int,
) (groups []*group.Group, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		p, err := ctx.ProfilesStorage.GetByID(ctx.Context(), userID)
		if err != nil {
			return err
		}

		var groupsMap map[group.GroupID]*group.Group
		if p.Type() == profile.TypeCoach {
			groupsMap, err = ctx.GroupStorage.ListByCoach(ctx.Context(), group.CoachID(userID), limit, offset)
		} else {
			groupsMap, err = ctx.GroupStorage.ListByTrainee(ctx.Context(), group.TraineeID(userID), limit, offset)
		}

		if err != nil {
			return err
		}
		groups = lo.Values(groupsMap)
		return nil
	})
	return
}
