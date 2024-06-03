package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"github.com/burenotti/go_health_backend/internal/adapter/api"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	"github.com/burenotti/go_health_backend/internal/app/messagebus"
	profileapp "github.com/burenotti/go_health_backend/internal/app/profile"
	"github.com/burenotti/go_health_backend/internal/config"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/leporo/sqlf"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg := config.MustLoad(configPath)
	logger := initLogger(cfg)

	bus := messagebus.New(logger)
	bus.Register(auth.EventCreated, func(event domain.Event) error {
		logger.Info("processed user created event")
		return nil
	})

	sqlf.SetDialect(sqlf.PostgreSQL)

	db, err := sql.Open("pgx", cfg.DB.DSN)
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	authorizer := &authapp.Authorizer{
		Cost:             bcrypt.DefaultCost,
		Secret:           cfg.JWT.Secret,
		AccessTokenTTL:   cfg.JWT.AccessTokenTTL,
		AuthorizationTTL: cfg.JWT.RefreshTokenTTL,
	}

	authService := authapp.NewService(authorizer, logger)
	profileService := profileapp.New(logger)

	server := api.NewServer(
		api.Addr(cfg.Server.Host, cfg.Server.Port),
		api.Logger(logger),
		api.DBContext(storage.DB{DB: db}),
		api.MessageBus(bus),
		api.AuthService(authService),
		api.ProfileService(profileService),
	)

	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error)

	go func() {
		defer close(errCh)
		errCh <- server.Start()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server was not shutdown gracefully", "error", err)
		}
	case err := <-errCh:
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Error("server closed with unexpected error", "error", err)
			}
		}
	}
	logger.Info("server shutdown")
}

func initLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler
	switch cfg.App.Env {
	case config.Development:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})
	case config.Production:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		})
	default:
		panic("invalid env")
	}

	return slog.New(handler)
}
