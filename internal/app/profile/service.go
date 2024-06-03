package profileapp

import (
	"context"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
	"log/slog"
	"time"
)

type Service struct {
	logger *slog.Logger
}

func New(
	logger *slog.Logger,
) *Service {
	return &Service{
		logger: logger,
	}
}

func (s *Service) CreateTrainee(
	ctx context.Context,
	userID string,
	firstName string,
	lastName string,
	birthDate *time.Time,
	uow *unitofwork.UnitOfWork[*AtomicContext],
) (trainee *profile.Trainee, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		trainee = profile.NewTrainee(userID, firstName, lastName, birthDate)
		if err := ctx.ProfileStorage.Add(ctx.Context(), trainee); err != nil {
			return err
		}

		return ctx.Commit()
	})
	return
}

func (s *Service) CreateCoach(
	ctx context.Context,
	userID string,
	firstName string,
	lastName string,
	birthDate *time.Time,
	yearsExperience int,
	bio string,
	uow *unitofwork.UnitOfWork[*AtomicContext],
) (coach *profile.Coach, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		coach = profile.NewCoach(userID, firstName, lastName, birthDate, yearsExperience, bio)
		if err := ctx.ProfileStorage.Add(ctx.Context(), coach); err != nil {
			return err
		}
		return ctx.Commit()
	})
	return
}

func (s *Service) GetProfileByID(
	ctx context.Context,
	userID string,
	uow *unitofwork.UnitOfWork[*AtomicContext],
) (p profile.Profile, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		var err error
		p, err = ctx.ProfileStorage.GetByID(ctx.Context(), userID)
		return err
	})
	return
}

func (s *Service) GetTraineeByID(
	ctx context.Context,
	userID string,
	uow *unitofwork.UnitOfWork[*AtomicContext],
) (*profile.Trainee, error) {
	p, err := s.GetProfileByID(ctx, userID, uow)
	if err != nil {
		return nil, err
	}

	if t, ok := p.(*profile.Trainee); ok {
		return t, nil
	} else {
		return nil, profile.ErrProfileNotFound
	}
}

func (s *Service) GetCoachByID(
	ctx context.Context,
	userID string,
	uow *unitofwork.UnitOfWork[*AtomicContext],
) (*profile.Coach, error) {
	p, err := s.GetProfileByID(ctx, userID, uow)
	if err != nil {
		return nil, err
	}

	if c, ok := p.(*profile.Coach); ok {
		return c, nil
	} else {
		return nil, profile.ErrProfileNotFound
	}
}
