package profilestorage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/pgutil"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
	"github.com/leporo/sqlf"
	"time"
)

type PostgresStorage struct {
	base *pgutil.BasePostgresStorage
}

func NewPostgresStorage(db storage.DBContext) *PostgresStorage {
	return &PostgresStorage{
		base: pgutil.NewBasePostgresStorage(db),
	}
}

func (s *PostgresStorage) Add(ctx context.Context, p profile.Profile) error {
	switch v := p.(type) {
	case *profile.Coach:
		return s.AddTrainee(ctx, v)
	case *profile.Trainee:
		return s.AddCoach(ctx, v)
	default:
		panic("unknown profile type")
	}
}

func (s *PostgresStorage) AddCoach(ctx context.Context, t *profile.Trainee) error {
	q := sqlf.InsertInto("trainees_profiles").
		Set("user_id", t.UserID).
		Set("first_name", t.FirstName).
		Set("last_name", t.LastName).
		Set("birth_date", t.BirthDate)

	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "trainees_profiles_pkey") {
			return profile.ErrProfileExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) AddTrainee(ctx context.Context, c *profile.Coach) error {
	q := sqlf.InsertInto("coaches_profiles").
		Set("user_id", c.UserID).
		Set("first_name", c.FirstName).
		Set("last_name", c.LastName).
		Set("birth_date", c.BirthDate).
		Set("years_experience", c.YearsExperience).
		Set("bio", c.Bio)

	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "trainee_profiles_pkey") {
			return profile.ErrProfileExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) GetByID(ctx context.Context, userID string) (profile.Profile, error) {
	var r getByIDRow
	q := sqlf.PostgreSQL.From("users u").
		LeftJoin("coaches_profiles c", "u.user_id = c.user_id").
		LeftJoin("trainees_profiles t", "u.user_id = t.user_id").
		Where("u.user_id = ?", userID).
		Select("t.user_id AS trainee_id").To(&r.TraineeID).
		Select("t.first_name").To(&r.TraineeFirstName).
		Select("t.last_name").To(&r.TraineeLastName).
		Select("t.birth_date").To(&r.TraineeBirthDate).
		Select("c.user_id AS coach_id").To(&r.CoachID).
		Select("c.first_name").To(&r.CoachFirstName).
		Select("c.last_name").To(&r.CoachLastName).
		Select("c.birth_date").To(&r.CoachBirthDate).
		Select("c.years_experience").To(&r.CoachYearsExperience).
		Select("c.bio").To(&r.CoachBio)

	if err := q.QueryRowAndClose(ctx, s.base.DB); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, profile.ErrProfileNotFound
		}
		return nil, err
	}

	if r.CoachID != nil {
		return &profile.Coach{
			UserID:          *r.CoachID,
			FirstName:       *r.CoachFirstName,
			LastName:        *r.CoachLastName,
			BirthDate:       r.CoachBirthDate,
			YearsExperience: *r.CoachYearsExperience,
			Bio:             *r.CoachBio,
		}, nil
	}

	if r.TraineeID != nil {
		return &profile.Trainee{
			UserID:    *r.TraineeID,
			FirstName: *r.TraineeFirstName,
			LastName:  *r.TraineeLastName,
			BirthDate: r.TraineeBirthDate,
		}, nil
	}

	return nil, profile.ErrProfileNotFound
}

func (s *PostgresStorage) Persist(ctx context.Context, p profile.Profile) error {
	switch v := p.(type) {
	case *profile.Coach:
		return s.PersistCoach(ctx, v)
	case *profile.Trainee:
		return s.PersistTrainee(ctx, v)
	default:
		panic("unknown profile type")
	}
}

func (s *PostgresStorage) PersistTrainee(ctx context.Context, t *profile.Trainee) error {
	q := sqlf.Update("trainees_profiles").
		Where("user_id = ?", t.UserID).
		Set("first_name", t.FirstName).
		Set("last_name", t.LastName).
		Set("last_name", t.BirthDate)

	res, err := q.ExecAndClose(ctx, s.base.DB)
	return pgutil.AssertUpdated(res, err, profile.ErrProfileNotFound)
}

func (s *PostgresStorage) PersistCoach(ctx context.Context, c *profile.Coach) error {
	q := sqlf.Update("coaches_profiles").
		Where("user_id = ?", c.UserID).
		Set("first_name", c.FirstName).
		Set("last_name", c.LastName).
		Set("last_name", c.BirthDate).
		Set("bio", c.Bio).
		Set("years_experience", c.YearsExperience)

	res, err := q.ExecAndClose(ctx, s.base.DB)
	return pgutil.AssertUpdated(res, err, profile.ErrProfileNotFound)
}

func (s *PostgresStorage) CollectEvents() []domain.Event {
	return s.base.CollectEvents()
}

func (s *PostgresStorage) Close() error {
	return nil
}

type getByIDRow struct {
	CoachID              *string
	CoachFirstName       *string
	CoachLastName        *string
	CoachBirthDate       *time.Time
	CoachYearsExperience *int
	CoachBio             *string

	TraineeID        *string
	TraineeFirstName *string
	TraineeLastName  *string
	TraineeBirthDate *time.Time
}
