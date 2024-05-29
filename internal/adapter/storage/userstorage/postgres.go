package userstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/burenotti/go_health_backend/internal/adapter/storage"
	"github.com/burenotti/go_health_backend/internal/domain"
	"github.com/burenotti/go_health_backend/internal/domain/auth"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leporo/sqlf"
	"github.com/r3labs/diff"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrInternal = errors.New("internal storage error")
)

type PostgresStorage struct {
	db     storage.DBContext
	logger *slog.Logger
	seenMu sync.Mutex
	seen   map[string]*auth.User
}

func NewPostgresStorage(db storage.DBContext, logger *slog.Logger) *PostgresStorage {

	return &PostgresStorage{
		db:     db,
		logger: logger,
		seen:   make(map[string]*auth.User),
		seenMu: sync.Mutex{},
	}
}

func (s *PostgresStorage) Add(ctx context.Context, u *auth.User) error {
	q := sqlf.InsertInto("users").
		Set("user_id", u.UserID).
		Set("email", u.Email).
		Set("password_hash", u.PasswordHash).
		Set("created_at", u.CreatedAt).
		Set("updated_at", u.UpdatedAt)

	if _, err := q.Exec(ctx, s.db); err != nil {
		if isUserDuplicated(err) {
			return errors.Join(fmt.Errorf("user exists: %w", err), auth.ErrUserExists)
		}
		return internalError(err)
	}

	for _, auth := range u.Authorizations {
		if err := s.addAuth(ctx, u.UserID, &auth); err != nil {
			return err
		}
	}

	s.markSeen(u)

	return nil
}

func (s *PostgresStorage) addAuth(ctx context.Context, userId string, a *auth.Authorization) error {
	addAuth := sqlf.InsertInto("authorizations").
		Set("identifier", a.Identifier).
		Set("logout_at", a.LogoutAt).
		Set("created_at", a.CreatedAt).
		Set("valid_until", a.ValidUntil).
		Set("user_id", userId)

	addDevice := sqlf.InsertInto("devices").
		Set("authorization_identifier", a.Identifier).
		Set("os", a.Device.OS).
		Set("device_model", a.Device.Model).
		Set("ip_address", a.Device.IPAddress).
		Set("browser", a.Device.Browser)

	if _, err := addAuth.Exec(ctx, s.db); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return auth.ErrAuthorizationExists
		}

		return internalError(err)
	}

	if _, err := addDevice.Exec(ctx, s.db); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return auth.ErrDeviceExists
		}

		return internalError(err)
	}

	return nil
}

func (s *PostgresStorage) get(
	ctx context.Context,
	whereClause string,
	whereArgs ...any,
) (users []*auth.User, outErr error) {

	var tmp userWithAuthRow

	q := sqlf.From("users u").
		LeftJoin("authorizations a", "u.user_id = a.user_id").
		LeftJoin("devices d", "a.identifier = d.authorization_identifier").
		Where(whereClause, whereArgs...).
		Select("u.user_id").To(&tmp.UserID).
		Select("u.email").To(&tmp.Email).
		Select("u.password_hash").To(&tmp.PasswordHash).
		Select("u.created_at").To(&tmp.CreatedAt).
		Select("u.updated_at").To(&tmp.UpdatedAt).
		Select("a.identifier").To(&tmp.Identifier).
		Select("a.valid_until").To(&tmp.AuthValidUntil).
		Select("a.logout_at").To(&tmp.LogoutAt).
		Select("a.created_at").To(&tmp.AuthCreatedAt).
		Select("d.os").To(&tmp.OS).
		Select("d.browser").To(&tmp.Browser).
		Select("d.device_model").To(&tmp.Model).
		Select("d.ip_address").To(&tmp.IpAddress)

	var fetchedRows []userWithAuthRow

	err := q.Query(ctx, s.db, func(rows *sql.Rows) {
		fetchedRows = append(fetchedRows, tmp)
	})

	if err != nil {
		return nil, internalError(err)
	}

	return rowsToDomain(fetchedRows), outErr
}

func (s *PostgresStorage) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	users, err := s.get(ctx, "u.email = ? ", email)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, auth.ErrUserNotFound
	}
	return users[0], nil
}

func (s *PostgresStorage) GetByID(ctx context.Context, userId string) (*auth.User, error) {
	users, err := s.get(ctx, "u.user_id = ? ", userId)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, auth.ErrUserNotFound
	}
	return users[0], nil
}

func (s *PostgresStorage) GetByAuthorization(ctx context.Context, identifier string) (*auth.User, error) {
	users, err := s.get(ctx, "a.identifier = ? ", identifier)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, auth.ErrUserNotFound
	}
	return users[0], nil
}

func (s *PostgresStorage) Persist(ctx context.Context, u *auth.User) error {
	dbState, err := s.GetByID(ctx, u.UserID)
	if err != nil {
		return err
	}

	if log, _ := diff.Diff(u, dbState); len(log) != 0 {
		q := sqlf.Update("users").Where("user_id = ?", u.UserID)
		q, err := makeUpdateQuery(*q, log)
		if err != nil {
			return err
		}

		res, err := q.Exec(ctx, s.db)
		if err != nil {
			return internalError(err)
		}

		affected, affErr := res.RowsAffected()
		if affErr != nil {
			return internalError(err)
		}

		if affected == 0 {
			return fmt.Errorf("can't persist auth data: %w", auth.ErrUserNotFound)
		}
	}

	dbAuthSet := make(map[string]*auth.Authorization)
	for _, auth := range dbState.Authorizations {
		dbAuthSet[auth.Identifier] = &auth
	}

	for _, auth := range u.Authorizations {
		if _, ok := dbAuthSet[auth.Identifier]; !ok {
			if err := s.addAuth(ctx, u.UserID, &auth); err != nil {
				return err
			}
		} else {
			if err := s.persistAuth(ctx, dbAuthSet[auth.Identifier], &auth); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *PostgresStorage) CollectEvents() []domain.Event {
	var events []domain.Event
	for _, u := range s.seen {
		events = append(events, u.PopEvents()...)
	}
	s.clearSeen()
	return events
}

func (s *PostgresStorage) Close() error {
	return nil
}

func (s *PostgresStorage) markSeen(u *auth.User) {
	s.seenMu.Lock()
	s.seen[u.UserID] = u
	s.seenMu.Unlock()
}
func (s *PostgresStorage) clearSeen() {
	s.seenMu.Lock()
	s.seen = make(map[string]*auth.User)
	s.seenMu.Unlock()
}

func (s *PostgresStorage) persistAuth(ctx context.Context, source, changed *auth.Authorization) error {
	log, _ := diff.Diff(source, changed)
	if len(log) == 0 {
		return s.persistDevice(ctx, source.Identifier, &source.Device, &changed.Device)
	}
	q := sqlf.Update("authorizations").Where("identifier = ?", source.Identifier)
	q, err := makeUpdateQuery(*q, log)
	if err != nil {
		return internalError(err)
	}

	if _, err := q.Exec(ctx, s.db); err != nil {
		return internalError(err)
	}
	return s.persistDevice(ctx, source.Identifier, &source.Device, &changed.Device)
}

func (s *PostgresStorage) persistDevice(ctx context.Context, id string, source, changed *auth.Device) error {
	log, _ := diff.Diff(source, changed)
	if len(log) == 0 {
		return nil
	}

	q := sqlf.Update("devices").Where("authorization_identifier = ?", id)
	q, err := makeUpdateQuery(*q, log)
	if err != nil {
		return internalError(err)
	}

	if _, err := q.Exec(ctx, s.db); err != nil {
		return internalError(err)
	}
	return nil
}

func isUserDuplicated(err error) bool {
	pgErr := &pgconn.PgError{}
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) && pgErr.ConstraintName == "users_pkey"
}

func internalError(err error) error {
	return errors.Join(fmt.Errorf("internal storage error: %w", err), ErrInternal)
}

type userWithAuthRow struct {
	UserID       string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Identifier     *string
	LogoutAt       *time.Time
	AuthCreatedAt  *time.Time
	AuthValidUntil *time.Time

	IpAddress *string
	Browser   *string
	OS        *string
	Model     *string
}

func rowsToDomain(rows []userWithAuthRow) []*auth.User {
	usersMap := make(map[string]*auth.User)

	for _, row := range rows {
		if _, ok := usersMap[row.UserID]; !ok {
			usersMap[row.UserID] = &auth.User{
				UserID:         row.UserID,
				Email:          row.Email,
				PasswordHash:   row.PasswordHash,
				CreatedAt:      time.Time{},
				UpdatedAt:      time.Time{},
				Authorizations: make([]auth.Authorization, 0),
			}
		}
		if row.Identifier != nil {
			auth := auth.Authorization{
				Identifier: *row.Identifier,
				CreatedAt:  *row.AuthCreatedAt,
				ValidUntil: *row.AuthValidUntil,
				LogoutAt:   row.LogoutAt,
				Device: auth.Device{
					Browser:   *row.Browser,
					OS:        *row.OS,
					IPAddress: *row.IpAddress,
				},
			}
			usersMap[row.UserID].Authorizations = append(usersMap[row.UserID].Authorizations, auth)
		}
	}

	users := make([]*auth.User, 0, len(usersMap))

	for _, u := range usersMap {
		users = append(users, u)
	}
	return users
}

func domainToRows(user *auth.User) (res []userWithAuthRow) {
	for _, auth := range user.Authorizations {
		t := userWithAuthRow{
			UserID:         user.UserID,
			Email:          user.Email,
			PasswordHash:   user.PasswordHash,
			CreatedAt:      user.CreatedAt,
			UpdatedAt:      user.UpdatedAt,
			Identifier:     &auth.Identifier,
			LogoutAt:       auth.LogoutAt,
			AuthCreatedAt:  &auth.CreatedAt,
			AuthValidUntil: &auth.ValidUntil,
			IpAddress:      &auth.Device.IPAddress,
			Browser:        &auth.Device.Browser,
			OS:             &auth.Device.OS,
			Model:          &auth.Device.Model,
		}
		res = append(res, t)
	}
	return res
}

func makeUpdateQuery(base sqlf.Stmt, updates diff.Changelog) (*sqlf.Stmt, error) {
	res := &base

	for _, upd := range updates {
		if upd.Type != "update" {
			return nil, errors.New("invalid update type " + upd.Type)
		}
		if len(upd.Path) > 1 {
			return nil, errors.New("cannot process updates in nested structures")
		}

		res = res.Set(upd.Path[0], upd.To)
	}
	return res, nil
}
