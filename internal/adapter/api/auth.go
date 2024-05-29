package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/auth"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/user"
	"github.com/labstack/echo/v4"
	"github.com/mileusna/useragent"
	"net/http"
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

	device := user.Device{
		Browser:   agent.Name,
		OS:        agent.OS,
		IPAddress: ipAddress,
		Model:     agent.Device,
	}

	uow := unitofwork.New[*auth.AtomicContext](s.db, auth.NewAtomicContext, s.msgBus, s.logger)

	tokens, err := s.authService.Login(c.Request().Context(), uow, device, b.Email, b.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
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

	uow := unitofwork.New[*auth.AtomicContext](s.db, auth.NewAtomicContext, s.msgBus, s.logger)

	ctx := c.Request().Context()
	_, err := s.authService.CreateUser(ctx, uow, b.UserID, b.Email, b.Password)
	if err != nil {
		if errors.Is(err, user.ErrUserExists) {
			return JsonError(c, http.StatusBadRequest, "user already exists")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusCreated)
}

func (s *Server) Logout(c echo.Context) error {
	u := c.Get(KeyCurrentUser).(*auth.AccessTokenData)

	uow := unitofwork.New[*auth.AtomicContext](s.db, auth.NewAtomicContext, s.msgBus, s.logger)
	if err := s.authService.Logout(c.Request().Context(), uow, u.UserID, u.Authorization); err != nil {
		if errors.Is(err, user.ErrUnauthorized) {
			return JsonError(c, http.StatusUnauthorized, "unauthorized")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) Refresh(ctx echo.Context) error {
	panic("implement me")
}
