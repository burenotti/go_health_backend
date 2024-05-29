package api

import (
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/app/auth"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"log/slog"
	"net"
	"strconv"
)

type Option func(*Server)

func Addr(host string, port int) Option {
	return func(s *Server) {
		s.addr = net.JoinHostPort(host, strconv.Itoa(port))
	}
}

func Logger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

func DBContext(db storage.DB) Option {
	return func(s *Server) {
		s.db = db
	}
}

func AuthService(service *auth.Service) Option {
	return func(s *Server) {
		s.authService = service
	}
}

func MessageBus(bus unitofwork.MessageBus) Option {
	return func(s *Server) {
		s.msgBus = bus
	}
}
