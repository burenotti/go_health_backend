package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	groupservice "github.com/burenotti/go_health_backend/internal/app/group"
	inviteservice "github.com/burenotti/go_health_backend/internal/app/invite"
	metricservice "github.com/burenotti/go_health_backend/internal/app/metric"
	profileapp "github.com/burenotti/go_health_backend/internal/app/profile"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	slogecho "github.com/samber/slog-echo"
	"log/slog"
	"time"
)

type Server struct {
	handler        *echo.Echo
	logger         *slog.Logger
	addr           string
	db             storage.DB
	authService    *authapp.Service
	profileService *profileapp.Service
	groupService   *groupservice.Service
	inviteService  *inviteservice.Service
	metricService  *metricservice.Service
	msgBus         unitofwork.MessageBus
	validator      *validator.Validate
}

func NewServer(opt ...Option) *Server {
	e := echo.New()

	// TODO: что-то сделать с магическими константами
	e.Server.WriteTimeout = 10 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.IdleTimeout = 10 * time.Second
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.MaxHeaderBytes = 4096

	v := validator.New(validator.WithRequiredStructEnabled())

	s := &Server{
		handler:   e,
		validator: v,
	}

	for _, opt := range opt {
		opt(s)
	}

	e.Use(slogecho.NewWithConfig(s.logger, slogecho.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelInfo,
		ServerErrorLevel: slog.LevelError,
		WithRequestID:    true,
		WithSpanID:       true,
		WithTraceID:      true,
	}))
	//e.Use(middleware.Recover())
	s.Mount()
	return s
}

func (s *Server) Mount() {
	s.MountAuth()
	s.MountProfile()
	s.MountGroups()
	s.MountInvites()
	s.MountMetrics()
}

func (s *Server) Start() error {
	return s.handler.Start(s.addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.handler.Shutdown(ctx)
}

func (s *Server) bind(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return fmt.Errorf("bad request")
	}
	if err := s.validator.Struct(i); err != nil {
		var errs validator.ValidationErrors
		if !errors.As(err, &errs) {
			return fmt.Errorf("bad request")
		}
		return fmt.Errorf("%s: %s", errs[0].Field(), errs[0].Error())

	}
	return nil
}
