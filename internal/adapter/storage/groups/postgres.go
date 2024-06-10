package groupstorage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/pgutil"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/group"
	"github.com/leporo/sqlf"
	"log/slog"
)

type PostgresStorage struct {
	base   *pgutil.BasePostgresStorage
	logger *slog.Logger
}

func NewPostgresStorage(db storage.DBContext, logger *slog.Logger) *PostgresStorage {
	return &PostgresStorage{
		base:   pgutil.NewBasePostgresStorage(db),
		logger: logger,
	}
}

func (s *PostgresStorage) Add(ctx context.Context, g *group.Group) error {
	q := sqlf.InsertInto("groups").
		Set("group_id", g.GroupID).
		Set("name", g.Name).
		Set("description", g.Description).
		Set("coach_id", g.CoachID).
		Set("created_at", g.CreatedAt).
		Set("updated_at", g.UpdatedAt)

	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "groups_pkey") {
			return group.ErrGroupExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) get(
	ctx context.Context,
	modify func(stmt *sqlf.Stmt) *sqlf.Stmt,
) (map[group.GroupID]*group.Group, error) {
	tmp := &group.Group{}

	q := sqlf.From("groups g").
		Select("g.group_id").To(&tmp.GroupID).
		Select("g.name").To(&tmp.Name).
		Select("g.description").To(&tmp.Description).
		Select("g.coach_id").To(&tmp.CoachID).
		Select("g.created_at").To(&tmp.CreatedAt).
		Select("g.updated_at").To(&tmp.UpdatedAt)

	q = modify(q)

	groups := make(map[group.GroupID]*group.Group)

	err := q.Query(ctx, s.base.DB, func(rows *sql.Rows) {

		groups[tmp.GroupID] = &group.Group{
			GroupID:     tmp.GroupID,
			Name:        tmp.Name,
			Description: tmp.Description,
			CoachID:     tmp.CoachID,
			CreatedAt:   tmp.CreatedAt,
			UpdatedAt:   tmp.UpdatedAt,
		}
	})

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return groups, err
}

func (s *PostgresStorage) ListByCoach(
	ctx context.Context,
	coachID group.CoachID,
	limit int,
	offset int,
) (map[group.GroupID]*group.Group, error) {
	return s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.Where("g.coach_id = ?", coachID).Offset(offset).Limit(limit)
	})
}

func (s *PostgresStorage) ListByTrainee(
	ctx context.Context,
	traineeID group.TraineeID,
	limit int,
	offset int,
) (map[group.GroupID]*group.Group, error) {
	return s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.LeftJoin("invites i", "g.group_id = i.group_id").
			LeftJoin("invites_accept ia", "i.invite_id = ia.invite_id").
			Where("ia.trainee_id = ?", traineeID).
			Offset(offset).
			Limit(limit)
	})
}

func (s *PostgresStorage) GetByID(ctx context.Context, groupID group.GroupID) (*group.Group, error) {
	g, err := s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.Where("g.group_id = ?", groupID)
	})

	return pgutil.PeekOrErr(g, err, group.ErrGroupNotFound)
}

func (s *PostgresStorage) GetMembers(
	ctx context.Context,
	groupId group.GroupID,
	limit, offset int,
) (result []*group.Member, err error) {

	var tmp struct {
		TraineeID string
		Email     string
		FirstName string
		LastName  string
	}

	q := sqlf.From("groups g").
		Join("invites i", "i.group_id = g.group_id").
		Join("invites_accept ia", "i.invite_id = ia.invite_id").
		Join("trainees_profiles t", "t.user_id = ia.trainee_id").
		Join("users u", "u.user_id = t.user_id").
		Where("g.group_id = ?", groupId).
		Limit(limit).
		Offset(offset).
		Select("t.user_id").To(&tmp.TraineeID).
		Select("u.email").To(&tmp.Email).
		Select("t.first_name").To(&tmp.FirstName).
		Select("t.last_name").To(&tmp.LastName)

	err = q.QueryAndClose(ctx, s.base.DB, func(rows *sql.Rows) {
		result = append(result, &group.Member{
			TraineeID: group.TraineeID(tmp.TraineeID),
			FirstName: tmp.FirstName,
			LastName:  tmp.LastName,
			Email:     tmp.Email,
		})
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return
}

func (s *PostgresStorage) Close() error {
	s.base.Close()
	return nil
}

func (s *PostgresStorage) CollectEvents() []domain.Event {
	return s.base.CollectEvents()
}
