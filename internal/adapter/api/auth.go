package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
	"github.com/labstack/echo/v4"
	"github.com/mileusna/useragent"
	"net/http"
	"strings"
)

func (s *Server) MountAuth() {
	loginRequired := LoginRequired(s.authService.Authorizer)

	authRoutes := s.handler.Group("/auth")

	authRoutes.POST("/login", s.Login)
	authRoutes.POST("/sign-up", s.SignUp)
	authRoutes.POST("/refresh", s.Refresh)
	authRoutes.POST("/logout", s.Logout, loginRequired)
}

type loginReq struct {
	Email    string `form:"username" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
}

type loginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) Login(c echo.Context) error {
	var b loginReq
	if err := s.bind(c, &b); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	agent := useragent.Parse(c.Request().UserAgent())

	ipAddress := c.Request().RemoteAddr
	if c.Request().Header.Get("X-Forwarded-For") != "" {
		ipAddress = c.Request().Header.Get("X-Forwarded-For")
	}

	device := auth.Device{
		Browser:   agent.Name,
		OS:        agent.OS,
		IPAddress: ipAddress,
		Model:     agent.Device,
	}

	uow := unitofwork.New[*authapp.AtomicContext](s.db, authapp.NewAtomicContext, s.msgBus, s.logger)

	tokens, err := s.authService.Login(c.Request().Context(), uow, device, b.Email, b.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return JsonError(c, http.StatusUnauthorized, "invalid email or password")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, &loginResp{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

type signUpReq struct {
	UserID   string `json:"user_id" validate:"required,uuid"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

func (s *Server) SignUp(c echo.Context) error {
	var b signUpReq
	if err := s.bind(c, &b); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := unitofwork.New[*authapp.AtomicContext](s.db, authapp.NewAtomicContext, s.msgBus, s.logger)

	ctx := c.Request().Context()
	_, err := s.authService.CreateUser(ctx, uow, b.UserID, b.Email, b.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return JsonError(c, http.StatusBadRequest, "user already exists")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusCreated)
}

func (s *Server) Logout(c echo.Context) error {
	u := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)

	uow := unitofwork.New[*authapp.AtomicContext](s.db, authapp.NewAtomicContext, s.msgBus, s.logger)
	if err := s.authService.Logout(c.Request().Context(), uow, u.UserID, u.Authorization); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return JsonError(c, http.StatusUnauthorized, "unauthorized")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}

type refreshResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) Refresh(c echo.Context) error {
	header := c.Request().Header.Get("Authorization")
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		return JsonError(c, http.StatusBadRequest, "invalid authorization header")
	}

	if parts[0] != "Refresh" {
		return JsonError(c, http.StatusBadRequest, "invalid authorization header")
	}

	uow := unitofwork.New[*authapp.AtomicContext](s.db, authapp.NewAtomicContext, s.msgBus, s.logger)
	ctx := c.Request().Context()
	tokens, err := s.authService.Refresh(ctx, uow, parts[1])
	if err != nil {
		if errors.Is(err, authapp.ErrInvalidAuthorization) {
			return JsonError(c, http.StatusUnauthorized, err)
		}
	}

	return c.JSON(http.StatusOK, &refreshResp{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}
