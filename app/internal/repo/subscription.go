package repo

// TODO: проверить порядки импорта везде

import (
	"context"
	"fmt"
	"strings"
	"task_test/internal/db"
	"task_test/internal/logger"

	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SubInfo struct {
	ServiceName string    `json:"service_name"`
	MonthlyFee  int       `json:"monthly_fee"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date"`
}

type SubData struct {
	sqlDB db.DB
}

var _ SubRepo = &SubData{}

func (sd *SubData) Close() {
	sd.sqlDB.Close()
}

func NewSubRepo(dsn string, slowThreshold time.Duration) *SubData {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.NewDB(ctx, dsn, slowThreshold)
	if err != nil {
		logger.Log.Fatal("failed to connect to db", zap.Error(err))
	}

	return &SubData{
		sqlDB: pool,
	}
}

func (sd *SubData) Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	end := startDate.AddDate(0, nMonths, 0)

	id := uuid.New()
	query := `INSERT INTO subscriptions (id, service_name, monthly_fee, user_id, start_date, end_date)
	VALUES ($1, $2, $3, $4, $5, $6)`

	err := sd.sqlDB.Exec(ctx, query, id, serviceName, monthlyFee, userID, startDate, end)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (sd *SubData) Read(subID uuid.UUID, format string) (SubInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT service_name, monthly_fee, user_id, start_date, end_date
	FROM subscriptions
	WHERE id = $1`

	var info SubInfo
	var start, end time.Time
	row := sd.sqlDB.QueryRow(ctx, query, subID)

	err := row.Scan(&info.ServiceName, &info.MonthlyFee, &info.UserID, &start, &end)
	if err != nil {
		return SubInfo{}, err
	}

	info.StartDate = start.Format(format)
	info.EndDate = end.Format(format)

	return info, nil
}

func (sd *SubData) Update(subID uuid.UUID, monthlyFee int, endDate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE subscriptions
	SET monthly_fee = $1, end_date = $2
	WHERE id = $3`

	return sd.sqlDB.Exec(ctx, query, monthlyFee, endDate, subID)
}

func (sd *SubData) Delete(subID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `DELETE FROM subscriptions WHERE id = $1`

	return sd.sqlDB.Exec(ctx, query, subID)
}

type PageCursor struct {
	LastStart time.Time
	LastID    uuid.UUID
	Limit     int
}

func (sd *SubData) List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string, cur PageCursor) ([]SubInfo, *PageCursor, error) {

	var (
		query strings.Builder
		args  []any
		n     = 0
	)

	_, err := query.WriteString(`SELECT service_name, monthly_fee, user_id, start_date, end_date, id
	FROM subscriptions
	WHERE daterange(start_date, end_date, '[]') && daterange($1, $2, '[]')`)
	if err != nil {
		return nil, nil, err
	}
	args = append(args, startDate, endDate)
	n = 2

	if serviceName != "" {
		n++
		_, err := query.WriteString(fmt.Sprintf(" AND service_name = $%d", n))
		if err != nil {
			return nil, nil, err
		}
		args = append(args, serviceName)
	}

	if userID != uuid.Nil {
		n++
		_, err := query.WriteString(fmt.Sprintf(" AND user_id = $%d", n))
		if err != nil {
			return nil, nil, err
		}
		args = append(args, userID)
	}

	if !cur.LastStart.IsZero() && cur.LastID != uuid.Nil {
		n++
		lastStartIndex := n
		n++
		lastIdIndex := n
		args = append(args, cur.LastStart, cur.LastID)

		_, err := query.WriteString(fmt.Sprintf(" AND (start_date > $%d OR (start_date = $%d AND id > $%d))", lastStartIndex, lastStartIndex, lastIdIndex))
		if err != nil {
			return nil, nil, err
		}
	}

	_, err = query.WriteString(" ORDER BY start_date ASC, id ASC")
	if err != nil {
		return nil, nil, err
	}

	limit := cur.Limit
	n++
	args = append(args, limit+1)
	_, err = query.WriteString(fmt.Sprintf(" LIMIT $%d", n))
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := sd.sqlDB.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	res := make([]SubInfo, 0)
	var (
		gotExtra  bool
		lastStart time.Time
		lastID    uuid.UUID
	)

	for rows.Next() {
		var info SubInfo
		var s, e time.Time
		var id uuid.UUID
		err := rows.Scan(&info.ServiceName, &info.MonthlyFee, &info.UserID, &s, &e, &id)
		if err != nil {
			return nil, nil, err
		}

		if len(res) == limit {
			gotExtra = true
			lastStart = s
			lastID = id
			break
		}

		info.StartDate = s.Format(format)
		info.EndDate = e.Format(format)

		res = append(res, info)
		lastStart = s
		lastID = id
	}

	next := new(PageCursor)
	if gotExtra {
		next = &PageCursor{
			LastStart: lastStart,
			LastID:    lastID,
			Limit:     limit,
		}
	}

	return res, next, nil
}

func (sd *SubData) GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error) {

	var (
		query strings.Builder
		args  []any
		n     = 0
	)

	_, err := query.WriteString(`SELECT COALESCE(SUM(COALESCE(s.monthly_fee, 0) *
		GREATEST(
			0,
			(
				(EXTRACT(YEAR FROM date_trunc('month', LEAST(s.end_date, ($2::date - interval '1 day')))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', LEAST(s.end_date, ($2::date - interval '1 day')))))
				- (EXTRACT(YEAR FROM date_trunc('month', GREATEST(s.start_date, $1))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', GREATEST(s.start_date, $1))))
				+ 1
			)
		)
	), 0) AS total_sum
	FROM subscriptions s
	WHERE daterange(start_date, end_date, '[]') && daterange($1, $2, '[]')`)
	if err != nil {
		return 0, err
	}
	args = append(args, startDate, endDate)
	n = 2

	if serviceName != "" {
		n++
		_, err := query.WriteString(fmt.Sprintf(" AND service_name = $%d", n))
		if err != nil {
			return 0, err
		}
		args = append(args, serviceName)
	}

	if userID != uuid.Nil {
		n++
		_, err := query.WriteString(fmt.Sprintf(" AND user_id = $%d", n))
		if err != nil {
			return 0, err
		}
		args = append(args, userID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := sd.sqlDB.QueryRow(ctx, query.String(), args...)
	var res int64
	err = row.Scan(&res)
	if err != nil {
		return 0, err
	}

	return res, nil
}
