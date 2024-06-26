package api

import (
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	groupservice "github.com/burenotti/go_health_backend/internal/app/group"
	inviteservice "github.com/burenotti/go_health_backend/internal/app/invite"
	metricservice "github.com/burenotti/go_health_backend/internal/app/metric"
	profileapp "github.com/burenotti/go_health_backend/internal/app/profile"
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

func AuthService(service *authapp.Service) Option {
	return func(s *Server) {
		s.authService = service
	}
}
func ProfileService(service *profileapp.Service) Option {
	return func(s *Server) {
		s.profileService = service
	}
}

func GroupService(service *groupservice.Service) Option {
	return func(s *Server) {
		s.groupService = service
	}
}

func InviteService(service *inviteservice.Service) Option {
	return func(s *Server) {
		s.inviteService = service
	}
}

func MetricService(service *metricservice.Service) Option {
	return func(s *Server) {
		s.metricService = service
	}
}

func MessageBus(bus unitofwork.MessageBus) Option {
	return func(s *Server) {
		s.msgBus = bus
	}
}
