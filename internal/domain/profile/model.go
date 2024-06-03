package profile

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/domain"
	"time"
)

var (
	ErrProfileExists   = errors.New("profile already exists")
	ErrProfileNotFound = errors.New("profile not found")
)

const (
	TypeTrainee = "trainee"
	TypeCoach   = "coach"
)

type Profile interface {
	Type() string
	ID() string
}

type Trainee struct {
	domain.Aggregate
	UserID    string
	FirstName string
	LastName  string
	BirthDate *time.Time
}

func NewTrainee(
	userID string,
	firstName string,
	lastName string,
	birthDate *time.Time,
) *Trainee {
	return &Trainee{
		UserID:    userID,
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: birthDate,
	}
}

func (t *Trainee) ID() string {
	return t.UserID
}

func (*Trainee) Type() string {
	return TypeTrainee
}

type Coach struct {
	domain.Aggregate
	UserID          string
	FirstName       string
	LastName        string
	BirthDate       *time.Time
	YearsExperience int
	Bio             string
}

func NewCoach(
	userID string,
	firstName string,
	lastName string,
	birthDate *time.Time,
	yearsExperience int,
	bio string,
) *Coach {
	return &Coach{
		UserID:          userID,
		FirstName:       firstName,
		LastName:        lastName,
		BirthDate:       birthDate,
		YearsExperience: yearsExperience,
		Bio:             bio,
	}
}

func (c *Coach) ID() string {
	return c.UserID
}

func (*Coach) Type() string {
	return TypeCoach
}
