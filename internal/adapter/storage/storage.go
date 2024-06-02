package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrInternal = errors.New("internal storage error")
)

type Comparable interface {
	Equals(other Comparable) bool
}

type Entity[ID Comparable] interface {
	ID() ID
}

type DBContext interface {
	Begin(ctx context.Context) (DBContext, error)
	Commit() error
	Rollback() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type DB struct {
	*sql.DB
}

func (D *DB) Commit() error {
	return nil
}

func (D *DB) Rollback() error {
	return nil
}

func (D *DB) Begin(ctx context.Context) (DBContext, error) {
	tx, err := D.DB.BeginTx(ctx, nil)
	return &Tx{tx}, err
}

type Tx struct {
	*sql.Tx
}

func (t *Tx) Begin(ctx context.Context) (DBContext, error) {
	return t, nil
}

func InternalError(err error) error {
	return errors.Join(fmt.Errorf("internal storage error: %w", err), ErrInternal)
}
