package metricservice

import (
	"context"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/metric"
	"log/slog"
)

type Service struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) CreateMetric(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	metricId, traineeId string,
	heartRate, weight, height int,
) error {
	return uow.Atomic(ctx, func(ctx *AtomicContext) error {
		m := metric.New(metricId, traineeId, heartRate, weight, height)

		if err := ctx.MetricStorage.Add(ctx.Context(), m); err != nil {
			return err
		}

		return ctx.Commit()
	})
}

func (s *Service) GetMetricByID(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	metricId string,
) (m *metric.Metric, outErr error) {
	outErr = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		var err error
		if m, err = ctx.MetricStorage.GetByID(ctx.Context(), metricId); err != nil {
			return err
		}

		return ctx.Commit()
	})
	return
}

func (s *Service) ListMetricByTrainee(
	ctx context.Context,
	uow *unitofwork.UnitOfWork[*AtomicContext],
	traineeId string,
) (m []*metric.Metric, outErr error) {
	outErr = uow.Atomic(ctx, func(ctx *AtomicContext) error {
		var err error
		if m, err = ctx.MetricStorage.ListByTrainee(ctx.Context(), traineeId); err != nil {
			return err
		}

		return ctx.Commit()
	})
	return
}
