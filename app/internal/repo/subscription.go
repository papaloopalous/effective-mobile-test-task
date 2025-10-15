package repo

// TODO: проверить порядки импорта везде

import (
	"context"
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
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT DO NOTHING`

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

func (sd *SubData) List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string) []SubInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT service_name, monthly_fee, user_id, start_date, end_date
	FROM subscriptions
	WHERE (service_name = $1 OR $1 = '')
		AND (user_id = $2 OR $2 = '00000000-0000-0000-0000-000000000000')
		AND start_date >= $3 AND end_date <= $4
	ORDER BY start_date ASC`

	rows, err := sd.sqlDB.Query(ctx, query, serviceName, userID, startDate, endDate)
	if err != nil {
		return []SubInfo{}
	}
	defer rows.Close()

	res := make([]SubInfo, 0)
	for rows.Next() {
		var info SubInfo
		var s, e time.Time
		err := rows.Scan(&info.ServiceName, &info.MonthlyFee, &info.UserID, &s, &e)
		if err != nil {
			continue
		}

		info.StartDate = s.Format(format)
		info.EndDate = e.Format(format)

		res = append(res, info)
	}

	return res
}

func (sd *SubData) GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT COALESCE(SUM(COALESCE(s.monthly_fee, 0) *
		GREATEST(
			0,
			(
				(EXTRACT(YEAR FROM date_trunc('month', LEAST(s.end_date, ($4::date - interval '1 day')))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', LEAST(s.end_date, ($4::date - interval '1 day')))))
				- (EXTRACT(YEAR FROM date_trunc('month', GREATEST(s.start_date, $3))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', GREATEST(s.start_date, $3))))
				+ 1
			)
		)
	), 0) AS total_sum
	FROM subscriptions s
	WHERE (s.service_name = $1 OR $1 = '')
		AND (s.user_id = $2 OR $2 = '00000000-0000-0000-0000-000000000000')
		AND s.start_date < $4 AND s.end_date >= $3;`

	row := sd.sqlDB.QueryRow(ctx, query, serviceName, userID, startDate, endDate)
	var res int64
	err := row.Scan(&res)
	if err != nil {
		return 0
	}

	return res
}
