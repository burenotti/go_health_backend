package metricstorage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/pgutil"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/metric"
	"github.com/leporo/sqlf"
	"github.com/samber/lo"
)

type PostgresStorage struct {
	base *pgutil.BasePostgresStorage
}

func NewPostgresStorage(db storage.DBContext) *PostgresStorage {
	return &PostgresStorage{
		base: pgutil.NewBasePostgresStorage(db),
	}
}

func (s *PostgresStorage) Add(ctx context.Context, m *metric.Metric) error {
	q := sqlf.InsertInto("metrics").
		Set("metric_id", m.MetricID).
		Set("trainee_id", m.TraineeID).
		Set("heart_rate", m.HeartRate).
		Set("weight", m.Weight).
		Set("height", m.Height).
		Set("created_at", m.CreatedAt)

	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "metrics_pkey") {
			return metric.ErrMetricExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) get(
	ctx context.Context,
	modify func(stmt *sqlf.Stmt),
) (map[string]*metric.Metric, error) {
	var tmp metric.Metric

	q := sqlf.From("metrics m").
		Select("m.metric_id").To(&tmp.MetricID).
		Select("m.trainee_id").To(&tmp.TraineeID).
		Select("m.heart_rate").To(&tmp.HeartRate).
		Select("m.weight").To(&tmp.Weight).
		Select("m.height").To(&tmp.Height).
		Select("m.created_at").To(&tmp.CreatedAt)

	modify(q)

	result := make(map[string]*metric.Metric)

	err := q.QueryAndClose(ctx, s.base.DB, func(rows *sql.Rows) {
		result[tmp.MetricID] = &metric.Metric{
			MetricID:  tmp.MetricID,
			TraineeID: tmp.TraineeID,
			HeartRate: tmp.HeartRate,
			Weight:    tmp.Weight,
			Height:    tmp.Height,
			CreatedAt: tmp.CreatedAt,
		}
	})

	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}

	return nil, err
}

func (s *PostgresStorage) GetByID(ctx context.Context, metricID string) (*metric.Metric, error) {
	result, err := s.get(ctx, func(stmt *sqlf.Stmt) {
		stmt.Where("m.metric_id = ?", metricID)
	})
	return pgutil.PeekOrErr(result, err, metric.ErrMetricNotFound)
}

func (s *PostgresStorage) ListByTrainee(ctx context.Context, traineeId string) ([]*metric.Metric, error) {
	result, err := s.get(ctx, func(stmt *sqlf.Stmt) {
		stmt.Where("m.trainee_id = ?", traineeId).OrderBy("m.created_at DESC")
	})
	if err != nil {
		return nil, err
	}
	return lo.Values(result), nil
}

func (s *PostgresStorage) CollectEvents() []domain.Event {
	return s.base.CollectEvents()
}

func (s *PostgresStorage) Close() error {
	s.base.Close()
	return nil
}
