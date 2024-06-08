package group

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/domain"
	"time"
)

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrGroupExists   = errors.New("group already exists")
)

type TraineeID string
type CoachID string
type InviteID string
type GroupID string

type Group struct {
	domain.Aggregate
	GroupID     GroupID   `diff:"group_id"`
	Name        string    `diff:"name"`
	Description string    `diff:"description"`
	CoachID     CoachID   `diff:"coach_id"`
	CreatedAt   time.Time `diff:"created_at"`
	UpdatedAt   time.Time `diff:"updated_at"`
}

func New(
	groupId GroupID,
	coachId CoachID,
	name string,
	description string,
) *Group {
	return &Group{
		GroupID:     groupId,
		Name:        name,
		Description: description,
		CoachID:     coachId,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

type Member struct {
	TraineeID TraineeID
	Email     string
	FirstName string
	LastName  string
}
