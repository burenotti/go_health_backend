package inviteservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/invite"
	"github.com/google/uuid"
	"log/slog"
)

type Service struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) CreateInvite(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	groupId invite.GroupID,
) (i *invite.Invite, err error) {
	err = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		inviteId := invite.InviteID(uuid.Must(uuid.NewUUID()).String())
		secret := s.generateSecret()
		i = invite.New(groupId, inviteId, secret)

		if err := ctx.InvitesStorage.Add(ctx.Context(), i); err != nil {
			return err
		}

		return ctx.Commit()
	})

	return i, err
}

func (s *Service) AcceptInvite(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	traineeId invite.TraineeID,
	secret string,
) error {
	return uow.Atomic(ctx, func(ctx *AtomicContext) error {
		inv, err := ctx.InvitesStorage.GetBySecret(ctx.Context(), secret)
		if err != nil {
			return err
		}

		if _, err := inv.AcceptInvite(traineeId, secret); err != nil {
			return err
		}

		if err := ctx.InvitesStorage.Persist(ctx.Context(), inv); err != nil {
			return err
		}

		return ctx.Commit()
	})
}

func (s *Service) generateSecret() string {
	var bytes [3]byte
	if n, err := rand.Read(bytes[:]); n != len(bytes) || err != nil {
		panic("failed to generate identifier")
	}

	return hex.EncodeToString(bytes[:])
}
