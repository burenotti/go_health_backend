package metricservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	metricstorage "github.com/burenotti/go_health_backend/internal/adapter/storage/metrics"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/metric"
)

type MetricStorage interface {
	Add(ctx context.Context, metric *metric.Metric) error
	GetByID(ctx context.Context, metricId string) (*metric.Metric, error)
	ListByTrainee(ctx context.Context, traineeId string) ([]*metric.Metric, error)
	CollectEvents() []domain.Event
	Close() error
}

type AtomicContext struct {
	ctx           context.Context
	db            storage.DBContext
	MetricStorage MetricStorage
}

func (a *AtomicContext) Context() context.Context {
	return a.ctx
}

func (a *AtomicContext) Commit() error {
	return a.db.Commit()
}

func (a *AtomicContext) Close() (err error) {
	if closeErr := a.MetricStorage.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	if err != nil {
		err = errors.Join(fmt.Errorf("failed to close storage"), err)
	}

	return err
}

func (a *AtomicContext) CollectEvents() []domain.Event {
	return a.MetricStorage.CollectEvents()
}

func NewAtomicContext(ctx context.Context, dbContext storage.DBContext) (*AtomicContext, error) {
	return &AtomicContext{
		ctx:           ctx,
		db:            dbContext,
		MetricStorage: metricstorage.NewPostgresStorage(dbContext),
	}, nil
}
