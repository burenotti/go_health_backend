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
	err = uow.Atomic(ctx, func(ctx context.Context, a *AtomicContext) error {
		u = auth.NewUser(userId, email, password, s.Authorizer)
		if err := a.UserStorage.Add(ctx, u); err != nil {
			return err
		}

		return a.Commit()
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
	err = uow.Atomic(ctx, func(ctx context.Context, a *AtomicContext) error {
		u, err := a.UserStorage.GetByEmail(ctx, email)

		if err != nil {
			return err
		}

		auth, err := u.Authorize(s.Authorizer, password, device)
		if err != nil {
			return err
		}

		accessToken, err := s.Authorizer.GenerateAccessToken(u, &auth)
		if err != nil {
			return err
		}

		if err := a.UserStorage.Persist(ctx, u); err != nil {
			return err
		}

		tokens = Tokens{
			AccessToken:  accessToken,
			RefreshToken: auth.Identifier,
		}
		return a.Commit()
	})
	return
}

func (s *Service) Logout(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	userId string,
	authIdentifier string,
) error {
	return uow.Atomic(ctx, func(ctx context.Context, a *AtomicContext) error {

		u, err := a.UserStorage.GetByID(ctx, userId)
		if err != nil {
			return err
		}

		if err := u.Logout(authIdentifier); err != nil {
			return err
		}

		if err := a.UserStorage.Persist(ctx, u); err != nil {
			return err
		}

		return a.Commit()
	})
}

func (s *Service) Refresh(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	authIdentifier string,
) (tokens Tokens, err error) {
	err = uow.Atomic(ctx, func(ctx context.Context, atomicContext *AtomicContext) error {
		user, err := atomicContext.UserStorage.GetByAuthorization(ctx, authIdentifier)
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
