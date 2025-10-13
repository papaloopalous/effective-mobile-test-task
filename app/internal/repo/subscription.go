package repo

import (
	"context"
	"log"
	"test_task/app/internal/db"
	"time"

	"github.com/google/uuid"
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
		log.Fatal("failed to connect to db: ", err)
	}

	return &SubData{
		sqlDB: pool,
	}
}

func (sd *SubData) Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate string, nMonths int) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start, err := time.Parse("01-2006", startDate)
	if err != nil {
		return uuid.Nil, err
	}
	end := start.AddDate(0, nMonths, 0)

	id := uuid.New()
	query := `INSERT INTO subscriptions (id, service_name, monthly_fee, user_id, start_date, end_date)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT DO NOTHING`

	err = sd.sqlDB.Exec(ctx, query, id, serviceName, monthlyFee, userID, start, end)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (sd *SubData) Read(subID uuid.UUID) (SubInfo, error) {
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

	info.StartDate = start.Format("01-2006")
	info.EndDate = end.Format("01-2006")

	return info, nil
}

func (sd *SubData) Update(subID uuid.UUID, monthlyFee int, endDate string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	end, err := time.Parse("01-2006", endDate)
	if err != nil {
		return err
	}

	query := `UPDATE subscriptions
	SET monthly_fee = $1, end_date = $2
	WHERE id = $3`

	return sd.sqlDB.Exec(ctx, query, monthlyFee, end, subID)
}

func (sd *SubData) Delete(subID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `DELETE FROM subscriptions WHERE id = $1`

	return sd.sqlDB.Exec(ctx, query, subID)
}

func (sd *SubData) List(serviceName string, userID uuid.UUID, startDate string, endDate string) []SubInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start, err := time.Parse("01-2006", startDate)
	if err != nil {
		return []SubInfo{}
	}

	end, err := time.Parse("01-2006", endDate)
	if err != nil {
		return []SubInfo{}
	}

	query := `SELECT service_name, monthly_fee, user_id, start_date, end_date
	FROM subscriptions
	WHERE service_name = $1 AND user_id = $2
		AND start_date >= $3 AND end_date <= $4
	ORDER BY start_date ASC`

	rows, err := sd.sqlDB.Query(ctx, query, serviceName, userID, start, end)
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

		info.StartDate = s.Format("01-2006")
		info.EndDate = e.Format("01-2006")

		res = append(res, info)
	}

	return res
}
func (sd *SubData) GetSum(serviceName string, userID uuid.UUID, startDate string, endDate string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start, err := time.Parse("01-2006", startDate)
	if err != nil {
		return 0
	}

	end, err := time.Parse("01-2006", endDate)
	if err != nil {
		return 0
	}

	query := `SELECT COALESCE(SUM(monthly_fee), 0)
	FROM subscriptions
	WHERE service_name = $1 AND user_id = $2
		AND start_date >= $3 AND end_date <= $4`

	row := sd.sqlDB.QueryRow(ctx, query, serviceName, userID, start, end)
	var res int64
	err = row.Scan(&res)
	if err != nil {
		return 0
	}

	return res
}
