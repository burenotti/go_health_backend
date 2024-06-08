package invite

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/domain"
	"time"
)

var (
	ErrInviteExists          = errors.New("invite already exists")
	ErrInviteExpired         = errors.New("invite expired")
	ErrInviteNotFound        = errors.New("invite not found")
	ErrInviteAlreadyAccepted = errors.New("invite already accepted")
	ErrInvalidSecret         = errors.New("invalid invite secret")
)

type InviteID string
type GroupID string
type TraineeID string

type Invite struct {
	domain.Aggregate
	InviteID   InviteID             `diff:"invite_id"`
	GroupID    GroupID              `diff:"-"`
	AcceptedBy map[TraineeID]Accept `diff:"-"`
	Secret     string               `diff:"secret"`
	CreatedAt  time.Time            `diff:"created_at"`
	ValidUntil time.Time            `diff:"valid_until"`
}

type Accept struct {
	InviteID   InviteID  `diff:"invite_id"`
	TraineeID  TraineeID `diff:"trainee_id"`
	AcceptedAt time.Time `diff:"accepted_at"`
}

func New(groupId GroupID, inviteId InviteID, secret string) *Invite {
	now := time.Now()

	invite := &Invite{
		InviteID:   inviteId,
		GroupID:    groupId,
		AcceptedBy: make(map[TraineeID]Accept),
		CreatedAt:  now,
		Secret:     secret,
		ValidUntil: now.Add(10 * time.Minute),
	}

	return invite
}

func (i *Invite) AcceptInvite(traineeID TraineeID, secret string) (Accept, error) {
	if accept, ok := i.AcceptedBy[traineeID]; ok {
		return accept, ErrInviteAlreadyAccepted
	}

	if i.Secret != secret {
		return Accept{}, ErrInvalidSecret
	}

	accept := Accept{
		InviteID:   i.InviteID,
		TraineeID:  traineeID,
		AcceptedAt: time.Now(),
	}

	i.AcceptedBy[traineeID] = accept
	return accept, nil
}
