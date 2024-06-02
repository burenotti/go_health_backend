package pgutil

import (
	"database/sql"
	"errors"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leporo/sqlf"
	"github.com/r3labs/diff"
	"sync"
)

type BasePostgresStorage struct {
	DB     storage.DBContext
	seenMu sync.Mutex
	seen   map[string]*auth.User
}

func NewBasePostgresStorage(db storage.DBContext) *BasePostgresStorage {
	return &BasePostgresStorage{
		DB: db,
	}
}

func (s *BasePostgresStorage) CollectEvents() []domain.Event {
	var events []domain.Event
	for _, u := range s.seen {
		events = append(events, u.PopEvents()...)
	}
	s.clearSeen()
	return events
}

func (s *BasePostgresStorage) Close() {
	s.clearSeen()
}

func (s *BasePostgresStorage) MarkSeen(u *auth.User) {
	s.seenMu.Lock()
	s.seen[u.UserID] = u
	s.seenMu.Unlock()
}
func (s *BasePostgresStorage) clearSeen() {
	s.seenMu.Lock()
	s.seen = make(map[string]*auth.User)
	s.seenMu.Unlock()
}

func ViolatesConstraint(err error, constraintName string) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) &&
		pgErr.ConstraintName == constraintName
}

func Peek[K comparable, V any](items map[K]V, defaultValue ...V) V {
	for _, item := range items {
		return item
	}

	if len(defaultValue) != 0 {
		return defaultValue[0]
	} else {
		return *new(V)
	}

}

func PeekOrErr[K comparable, V any](items map[K]V, err, notFoundErr error) (V, error) {

	if err != nil {
		return *new(V), err
	}

	if len(items) == 0 {
		return *new(V), notFoundErr
	}

	return Peek(items), nil
}

func MakeUpdateQuery(stmt *sqlf.Stmt, updates diff.Changelog) *sqlf.Stmt {

	for _, upd := range updates {
		if upd.Type != "update" {
			panic("invalid update type " + upd.Type)
		}
		if len(upd.Path) > 1 {
			panic("cannot process updates in nested structures")
		}

		stmt = stmt.Set(upd.Path[0], upd.To)
	}
	return stmt
}

func AssertUpdated(res sql.Result, err error, notUpdatedError error) error {
	if err != nil {
		return storage.InternalError(err)
	}

	affected, err := res.RowsAffected()

	if err != nil {
		return storage.InternalError(err)
	}

	if affected == 0 {
		return notUpdatedError
	}
	return nil
}
