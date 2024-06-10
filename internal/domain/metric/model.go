package metric

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/domain"
	"time"
)

var (
	ErrMetricExists    = errors.New("metric already exists")
	ErrMetricNotFound  = errors.New("metric not found")
	ErrTraineeNotFound = errors.New("trainee not found")
)

type Metric struct {
	domain.Aggregate
	MetricID  string
	TraineeID string
	HeartRate int
	Weight    int
	Height    int
	CreatedAt time.Time
}

func New(metricId, traineeId string, heartRate, weight, height int) *Metric {
	return &Metric{
		MetricID:  metricId,
		TraineeID: traineeId,
		HeartRate: heartRate,
		Weight:    weight,
		Height:    height,
		CreatedAt: time.Now().UTC(),
	}
}
