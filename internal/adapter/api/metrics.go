package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	metricservice "github.com/burenotti/go_health_backend/internal/app/metric"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/metric"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"net/http"
	"time"
)

func (s *Server) MountMetrics() {
	loginRequired := LoginRequired(s.authService.Authorizer)
	s.handler.POST("/metrics/:metric_id", s.CreateMetric, loginRequired)
	s.handler.GET("/metrics/:metric_id", s.GetMetric, loginRequired)
	s.handler.GET("/metrics/list/:trainee_id", s.ListMetrics, loginRequired)
}

func (s *Server) getMetricsUoW() *unitofwork.UnitOfWork[*metricservice.AtomicContext] {
	return unitofwork.New[*metricservice.AtomicContext](
		s.db,
		metricservice.NewAtomicContext,
		s.msgBus,
		s.logger,
	)
}

type CreateMetricRequest struct {
	MetricID  string `param:"metric_id"`
	HeartRate int    `json:"heart_rate"`
	Weight    int    `json:"weight"`
	Height    int    `json:"height"`
}

func (s *Server) CreateMetric(c echo.Context) error {
	var req CreateMetricRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	uow := s.getMetricsUoW()
	ctx := c.Request().Context()
	user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)

	err := s.metricService.CreateMetric(ctx, uow, req.MetricID, user.UserID, req.HeartRate, req.Weight, req.Height)
	if err != nil {
		if errors.Is(err, metric.ErrMetricExists) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusCreated)
}

type GetMetricRequest struct {
	MetricID string `param:"metric_id"`
}

type GetMetricResponse struct {
	MetricID  string    `json:"metric_id"`
	TraineeID string    `json:"trainee_id"`
	HeartRate int       `json:"heart_rate"`
	Weight    int       `json:"weight"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) GetMetric(c echo.Context) error {
	var req GetMetricRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	uow := s.getMetricsUoW()
	ctx := c.Request().Context()

	m, err := s.metricService.GetMetricByID(ctx, uow, req.MetricID)
	if err != nil {
		if errors.Is(err, metric.ErrMetricExists) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, GetMetricResponse{
		MetricID:  m.MetricID,
		TraineeID: m.TraineeID,
		HeartRate: m.HeartRate,
		Weight:    m.Weight,
		Height:    m.Height,
		CreatedAt: m.CreatedAt,
	})
}

type Metric struct {
	MetricID  string    `json:"metric_id"`
	TraineeID string    `json:"trainee_id"`
	HeartRate int       `json:"heart_rate"`
	Weight    int       `json:"weight"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
}

type ListMetricsRequest struct {
	TraineeID string `param:"trainee_id"`
}

type ListMetricsResponse struct {
	Metrics []Metric `json:"metrics"`
}

func (s *Server) ListMetrics(c echo.Context) error {
	var req ListMetricsRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	uow := s.getMetricsUoW()
	ctx := c.Request().Context()
	//user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)

	lst, err := s.metricService.ListMetricByTrainee(ctx, uow, req.TraineeID)
	if err != nil {
		if errors.Is(err, metric.ErrMetricExists) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, ListMetricsResponse{
		Metrics: lo.Map(lst, func(m *metric.Metric, _ int) Metric {
			return Metric{
				MetricID:  m.MetricID,
				TraineeID: m.TraineeID,
				HeartRate: m.HeartRate,
				Weight:    m.Weight,
				Height:    m.Height,
				CreatedAt: m.CreatedAt,
			}
		}),
	})
}
