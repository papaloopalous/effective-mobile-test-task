package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	Ping(ctx context.Context) error
	Exec(ctx context.Context, query string, args ...any) error
	Close()
}

type Pool struct {
	pool          *pgxpool.Pool
	slowThreshold time.Duration
}

func NewDB(ctx context.Context, dsn string, slowThreshold time.Duration) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	wrapper := &Pool{
		pool:          pool,
		slowThreshold: slowThreshold,
	}

	err = wrapper.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}

	return wrapper, nil
}

func (w *Pool) logQuery(method string, query string, args []any, elapsed time.Duration, rows int64, err error) {
	prefix := ""
	switch {
	case err != nil:
		prefix = "(ERROR)"
	case elapsed >= w.slowThreshold:
		prefix = "(SLOW)"
	default:
		prefix = "(OK)"
	}

	logMsg := "%s %s took %s with query %q"
	if len(args) > 0 {
		logMsg += " with args %v"
	}
	if rows > 0 {
		logMsg += " with affected rows %v"
	}
	if err != nil {
		logMsg += " with err %v"
	}

	log.Printf(logMsg, prefix, method, elapsed, query, args, rows, err)
}

func (w *Pool) Close() {
	w.pool.Close()
}

func (w *Pool) Ping(ctx context.Context) error {
	return w.pool.Ping(ctx)
}

func (w *Pool) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	start := time.Now()
	row := w.pool.QueryRow(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("QueryRow", query, args, elapsed, 0, nil)
	return row
}

func (w *Pool) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	start := time.Now()
	rows, err := w.pool.Query(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("Query", query, args, elapsed, 0, err)
	return rows, err
}

func (w *Pool) Exec(ctx context.Context, query string, args ...any) error {
	start := time.Now()
	tag, err := w.pool.Exec(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("Exec", query, args, elapsed, tag.RowsAffected(), err)
	return err
}
