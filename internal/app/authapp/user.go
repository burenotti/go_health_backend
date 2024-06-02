package authapp

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
	"log/slog"
)

var (
	ErrInvalidAuthorization = errors.New("invalid authorization")
)

type Service struct {
	logger     *slog.Logger
	Authorizer *Authorizer
}

func NewService(auth *Authorizer, logger *slog.Logger) *Service {
	return &Service{
		logger:     logger,
		Authorizer: auth,
	}
}

func (s *Service) CreateUser(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	userId string,
	email string,
	password string,
) (u *auth.User, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		u = auth.NewUser(userId, email, password, s.Authorizer)
		if err := ctx.UserStorage.Add(ctx.Context(), u); err != nil {
			return err
		}

		return ctx.Commit()
	})
	return
}

func (s *Service) Login(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	device auth.Device,
	email string,
	password string,
) (tokens Tokens, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		u, err := ctx.UserStorage.GetByEmail(ctx.Context(), email)

		if err != nil {
			return err
		}

		a, err := u.Authorize(s.Authorizer, password, device)
		if err != nil {
			return err
		}

		accessToken, err := s.Authorizer.GenerateAccessToken(u, &a)
		if err != nil {
			return err
		}

		if err := ctx.UserStorage.Persist(ctx.Context(), u); err != nil {
			return err
		}

		tokens = Tokens{
			AccessToken:  accessToken,
			RefreshToken: a.Identifier,
		}
		return ctx.Commit()
	})
	return
}

func (s *Service) Logout(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	userId string,
	authIdentifier string,
) error {
	return uow.Atomic(ctx, func(ctx *AtomicContext) error {

		u, err := ctx.UserStorage.GetByID(ctx.Context(), userId)
		if err != nil {
			return err
		}

		if err := u.Logout(authIdentifier); err != nil {
			return err
		}

		if err := ctx.UserStorage.Persist(ctx.Context(), u); err != nil {
			return err
		}

		return ctx.Commit()
	})
}

func (s *Service) Refresh(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	authIdentifier string,
) (tokens Tokens, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		user, err := ctx.UserStorage.GetByAuthorization(ctx.Context(), authIdentifier)
		if err != nil {
			return err
		}

		a := user.GetAuthorization(authIdentifier)
		if !a.IsActive() {
			return fmt.Errorf("%w: authorization is not active", ErrInvalidAuthorization)
		}

		tokens.AccessToken, err = s.Authorizer.GenerateAccessToken(user, a)
		tokens.RefreshToken = a.Identifier
		return err
	})
	return
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}
