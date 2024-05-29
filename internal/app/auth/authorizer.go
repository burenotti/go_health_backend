package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/domain/user"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrAccessTokenInvalid = errors.New("invalid access token")
	ErrAccessTokenExpired = fmt.Errorf("%w: token expired", ErrAccessTokenInvalid)
)

type Authorizer struct {
	Cost             int
	Secret           string
	AccessTokenTTL   time.Duration
	AuthorizationTTL time.Duration
}

func (a *Authorizer) Authorize(u *user.User, password string, dev user.Device) (user.Authorization, error) {
	hashBytes, err := hex.DecodeString(u.PasswordHash)
	if err != nil {
		return user.Authorization{}, err
	}

	if err := bcrypt.CompareHashAndPassword(hashBytes, []byte(password)); err != nil {
		return user.Authorization{}, user.ErrInvalidCredentials
	}

	now := time.Now().UTC()
	auth := user.Authorization{
		Identifier: a.generateIdentifier(),
		CreatedAt:  now,
		ValidUntil: now.Add(a.AuthorizationTTL),
		LogoutAt:   nil,
		Device:     dev,
	}
	return auth, nil
}

func (a *Authorizer) Hash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), a.Cost)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash)
}

func (a *Authorizer) generateIdentifier() string {
	var bytes [16]byte
	if n, err := rand.Read(bytes[:]); n != len(bytes) || err != nil {
		panic("failed to generate identifier")
	}

	return hex.EncodeToString(bytes[:])
}

func (a *Authorizer) GenerateAccessToken(u *user.User, auth *user.Authorization) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti": auth.Identifier,
		"sub": u.UserID,
		"exp": now.Add(a.AccessTokenTTL).Unix(),
		"iat": now.Unix(),
	})
	return token.SignedString([]byte(a.Secret))
}

type AccessTokenData struct {
	Authorization string
	UserID        string
}

func (a *Authorizer) ValidateAccessToken(accessToken string) (*AccessTokenData, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(accessToken, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.Secret), nil
	})

	if err != nil /* && err.(*jwt.ValidationError).Errors != jwt.ValidationErrorExpired */ {
		return nil, ErrAccessTokenInvalid
	}

	//if claims["exp"].(float64) < float64(time.Now().UTC().Unix()) {
	//	return nil, ErrAccessTokenExpired
	//}

	data := &AccessTokenData{
		Authorization: claims["jti"].(string),
		UserID:        claims["sub"].(string),
	}
	return data, err
}
