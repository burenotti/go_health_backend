package auth

import (
	"context"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/user"
	"log/slog"
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
) (u *user.User, err error) {
	err = uow.Atomic(ctx, func(ctx context.Context, a *AtomicContext) error {
		u = user.NewUser(userId, email, password, s.Authorizer)
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
	device user.Device,
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

type Tokens struct {
	AccessToken  string
	RefreshToken string
}
