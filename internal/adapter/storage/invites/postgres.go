package invitestorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/adapter/storage/pgutil"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/invite"
	"github.com/leporo/sqlf"
	"github.com/r3labs/diff"
	"log/slog"
	"time"
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

func (s *PostgresStorage) Add(ctx context.Context, inv *invite.Invite) error {
	q := sqlf.InsertInto("invites").
		Set("invite_id", inv.InviteID).
		Set("group_id", inv.GroupID).
		Set("secret", inv.Secret).
		Set("created_at", inv.CreatedAt).
		Set("valid_until", inv.ValidUntil)

	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "invites_pkey") {
			return invite.ErrInviteExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) get(
	ctx context.Context,
	modify func(stmt *sqlf.Stmt) *sqlf.Stmt,
) (map[invite.InviteID]*invite.Invite, error) {
	tmp := struct {
		InviteID   string
		GroupID    string
		Secret     string
		ValidUntil time.Time
		CreatedAt  time.Time
		TraineeID  *string
		AcceptedAt *time.Time
	}{}

	q := sqlf.From("invites i").
		LeftJoin("invites_accept a", "i.invite_id = a.invite_id").
		Select("i.invite_id").To(&tmp.InviteID).
		Select("i.group_id").To(&tmp.GroupID).
		Select("i.valid_until").To(&tmp.ValidUntil).
		Select("i.secret").To(&tmp.Secret).
		Select("i.created_at").To(&tmp.CreatedAt).
		Select("a.trainee_id").To(&tmp.TraineeID).
		Select("a.accepted_at").To(&tmp.AcceptedAt)

	q = modify(q)

	invites := make(map[invite.InviteID]*invite.Invite)

	err := q.Query(ctx, s.base.DB, func(rows *sql.Rows) {
		id := invite.InviteID(tmp.InviteID)
		if _, ok := invites[id]; !ok {
			invites[id] = &invite.Invite{
				InviteID:   id,
				GroupID:    invite.GroupID(tmp.GroupID),
				AcceptedBy: make(map[invite.TraineeID]invite.Accept),
				CreatedAt:  tmp.CreatedAt,
				ValidUntil: tmp.ValidUntil,
				Secret:     tmp.Secret,
			}
		}
		if tmp.TraineeID != nil {
			traineeId := invite.TraineeID(*tmp.TraineeID)
			invites[id].AcceptedBy[traineeId] = invite.Accept{
				InviteID:   id,
				TraineeID:  traineeId,
				AcceptedAt: *tmp.AcceptedAt,
			}
		}
	})

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return invites, err
}

func (s *PostgresStorage) Persist(ctx context.Context, inv *invite.Invite) error {
	dbState, err := s.GetByID(ctx, inv.InviteID)
	if err != nil {
		return err
	}
	log, err := diff.Diff(dbState, inv)
	if err != nil {
		panic(err) // should never happen
	}

	if len(log) != 0 {
		q := sqlf.Update("invites").Where("invite_id = ?", inv.InviteID)
		q = pgutil.MakeUpdateQuery(q, log)

		fmt.Println(q.String())
		res, err := q.ExecAndClose(ctx, s.base.DB)
		if err := pgutil.AssertUpdated(res, err, invite.ErrInviteNotFound); err != nil {
			return err
		}
	}
	for id, accept := range inv.AcceptedBy {
		var err error
		if _, ok := dbState.AcceptedBy[id]; !ok {
			err = s.AddAccept(ctx, accept)
		}
		return err
	}
	return nil
}

func (s *PostgresStorage) persistAccept(ctx context.Context, src, dirty invite.Accept) error {
	log, err := diff.Diff(src, dirty)
	if err != nil {
		panic(err) // should never happen
	}
	for _, upd := range log {
		if upd.Type == diff.CREATE {
			if err := s.AddAccept(ctx, dirty); err != nil {
				return err
			}
		} else {
			panic("invite accepts are not allowed to be modified or deleted")
		}
	}

	return nil
}

func (s *PostgresStorage) GetByID(ctx context.Context, inviteID invite.InviteID) (*invite.Invite, error) {
	invites, err := s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.Where("i.invite_id = ?", inviteID)
	})

	return pgutil.PeekOrErr(invites, err, invite.ErrInviteNotFound)
}

func (s *PostgresStorage) GetBySecret(ctx context.Context, secret string) (*invite.Invite, error) {
	invites, err := s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.Where("i.secret = ? AND i.valid_until >= ?", secret, time.Now().UTC())
	})

	return pgutil.PeekOrErr(invites, err, invite.ErrInviteNotFound)
}

func (s *PostgresStorage) ListByGroupID(
	ctx context.Context,
	groupID invite.GroupID,
) (map[invite.InviteID]*invite.Invite, error) {
	return s.get(ctx, func(stmt *sqlf.Stmt) *sqlf.Stmt {
		return stmt.Where("i.group_id = ?", groupID)
	})
}

func (s *PostgresStorage) AddAccept(
	ctx context.Context,
	accept invite.Accept,
) error {
	q := sqlf.InsertInto("invites_accept").
		Set("trainee_id", accept.TraineeID).
		Set("accepted_at", accept.AcceptedAt).
		Set("invite_id", accept.InviteID)

	fmt.Println(q.String())
	if _, err := q.ExecAndClose(ctx, s.base.DB); err != nil {
		if pgutil.ViolatesConstraint(err, "invites_accept_pkey") {
			return invite.ErrInviteAlreadyAccepted
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) Close() error {
	return nil
}

func (s *PostgresStorage) CollectEvents() []domain.Event {
	return s.base.CollectEvents()
}
