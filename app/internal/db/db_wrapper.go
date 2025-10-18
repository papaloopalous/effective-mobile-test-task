package db

import (
	"context"
	"time"

	"task_test/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DB interface {
	// QueryRow - выполнить запрос, ожидающий одну строку
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	// Query - выполнить запрос, возвращающий набор строк
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	// Ping - проверить соединение с БД
	Ping(ctx context.Context) error
	// Exec - выполнить команду без возврата набора строк
	Exec(ctx context.Context, query string, args ...any) error
	// Close - закрыть соединение/пул
	Close()
}

type Pool struct {
	pool          *pgxpool.Pool
	slowThreshold time.Duration
}

// NewDB - инициализирует пул соединений PostgreSQL с заданным порогом медленных запросов
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

// logQuery - служебная функция логирования запросов к БД
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

	logger.Log.Info("db query executed", zap.String("status", prefix), zap.String("method", method), zap.Duration("elapsed", elapsed), zap.String("query", query), zap.Any("args", args), zap.Int64("rows", rows), zap.Error(err))
}

// Close - закрывает пул соединений
func (w *Pool) Close() {
	w.pool.Close()
}

// Ping - проверяет доступность БД
func (w *Pool) Ping(ctx context.Context) error {
	return w.pool.Ping(ctx)
}

// QueryRow - выполняет запрос и возвращает одну строку
func (w *Pool) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	start := time.Now()
	row := w.pool.QueryRow(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("QueryRow", query, args, elapsed, 0, nil)
	return row
}

// Query - выполняет запрос и возвращает набор строк
func (w *Pool) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	start := time.Now()
	rows, err := w.pool.Query(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("Query", query, args, elapsed, 0, err)
	return rows, err
}

// Exec - выполняет команду и логирует количество затронутых строк
func (w *Pool) Exec(ctx context.Context, query string, args ...any) error {
	start := time.Now()
	tag, err := w.pool.Exec(ctx, query, args...)
	elapsed := time.Since(start)
	w.logQuery("Exec", query, args, elapsed, tag.RowsAffected(), err)
	return err
}
