package auth

import (
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/domain"
	"time"
)

type Kind string

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserExists          = errors.New("user already exists")
	ErrDeviceExists        = errors.New("device already exists")
	ErrAuthorizationExists = errors.New("authorization already exists")
	ErrUserEmailDuplicate  = fmt.Errorf("%w: email is not unique", ErrUserExists)
	ErrInvalidCredentials  = errors.New("email or password is invalid")
	ErrUnauthorized        = errors.New("unauthorized")
)

const (
	EventCreated  = "user.created"
	EventNewLogin = "user.login"
	EventLogout   = "user.logout"
)

type Authorizer interface {
	Hash(password string) string
	Authorize(u *User, password string, dev Device) (*Authorization, error)
}

type Device struct {
	Browser   string `diff:"browser"`
	OS        string `diff:"os"`
	IPAddress string `diff:"ip_address"`
	Model     string `diff:"model"`
}

type Authorization struct {
	ID         string     `diff:"-"`
	Secret     string     `diff:"-"`
	CreatedAt  time.Time  `diff:"-"`
	ValidUntil time.Time  `diff:"valid_until"`
	LogoutAt   *time.Time `diff:"logout_at"`
	Device     Device     `diff:"-"`
}

func (a *Authorization) IsActive() bool {
	return time.Now().Before(a.ValidUntil) && a.LogoutAt == nil
}

type User struct {
	domain.Aggregate `diff:"-"`
	UserID           string           `diff:"-"`
	Email            string           `diff:"email"`
	PasswordHash     string           `diff:"password_hash"`
	CreatedAt        time.Time        `diff:"-"`
	UpdatedAt        time.Time        `diff:"updated_at"`
	Authorizations   []*Authorization `diff:"-"`
}

func (u *User) GetAuthByID(authId string) *Authorization {
	for _, auth := range u.Authorizations {
		if auth.ID == authId {
			return auth
		}
	}
	return nil
}

func (u *User) GetAuthBySecret(secret string) *Authorization {
	for _, auth := range u.Authorizations {
		if auth.Secret == secret {
			return auth
		}
	}
	return nil
}

func NewUser(
	userID string,
	email,
	password string,
	hasher Authorizer,
) *User {
	u := &User{
		Aggregate:    domain.Aggregate{},
		UserID:       userID,
		Email:        email,
		PasswordHash: hasher.Hash(password),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	u.PushEvent(&CreatedEvent{
		At:     u.CreatedAt,
		UserID: u.UserID,
		Email:  u.Email,
	})
	return u
}

func (u *User) Authorize(a Authorizer, password string, dev Device) (*Authorization, error) {

	auth, err := a.Authorize(u, password, dev)
	if err != nil {
		return nil, err
	}

	u.Authorizations = append(u.Authorizations, auth)

	u.PushEvent(LoginEvent{
		At:     time.Now().UTC(),
		UserID: u.UserID,
		ID:     auth.Secret,
		Device: auth.Device,
	})

	return auth, nil
}

func (u *User) Logout(authId string) error {
	auth := u.GetAuthByID(authId)

	if auth == nil {
		return fmt.Errorf("%w: provided identifier not found", ErrUnauthorized)
	}

	if auth.LogoutAt != nil {
		return fmt.Errorf("%w: authorization already closed", ErrUnauthorized)
	}

	now := time.Now().UTC()
	auth.LogoutAt = &now

	u.PushEvent(LogoutEvent{
		At:     time.Now().UTC(),
		UserID: u.UserID,
		ID:     auth.ID,
	})

	return nil
}

type CreatedEvent struct {
	At        time.Time
	UserID    string
	Email     string
	FirstName string
	LastName  string
}

func (u CreatedEvent) Type() string {
	return EventCreated
}

func (u CreatedEvent) PublishedAt() time.Time {
	return u.At
}

type LoginEvent struct {
	At     time.Time
	UserID string
	ID     string
	Device Device
}

func (u LoginEvent) Type() string {
	return EventNewLogin
}

func (u LoginEvent) PublishedAt() time.Time {
	return u.At
}

type LogoutEvent struct {
	At     time.Time
	UserID string
	ID     string
}

func (u LogoutEvent) Type() string {
	return EventNewLogin
}

func (u LogoutEvent) PublishedAt() time.Time {
	return u.At
}
