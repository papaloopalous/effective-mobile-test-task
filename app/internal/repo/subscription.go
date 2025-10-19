package repo

import (
	"context"
	"strings"
	"time"

	"task_test/internal/db"
	"task_test/internal/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type SubRepo interface {
	// Create - создать подписку
	Create(ctx context.Context, args CreateArgs) (uuid.UUID, error)
	// Read - получить информацию о подписке по ID
	Read(ctx context.Context, subID uuid.UUID, format string) (SubInfo, error)
	// Update - обновить стоимость/дату окончания подписки
	Update(ctx context.Context, subID uuid.UUID, monthlyFee int, endDate time.Time) error
	// Delete - удалить подписку
	Delete(ctx context.Context, subID uuid.UUID) error
	// List - получить список подписок с фильтрами и курсором
	List(ctx context.Context, args ListArgs) ([]SubInfo, *PageCursor, error)
	// GetSum - получить суммарную стоимость по фильтрам
	GetSum(ctx context.Context, serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error)
	// Close - закрыть подключение/ресурсы
	Close()
}

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

// Close - закрывает соединение с БД/ресурсы пула
func (sd *SubData) Close() {
	sd.sqlDB.Close()
}

// NewSubRepo - создаёт новый репозиторий подписок, устанавливая соединение с БД
func NewSubRepo(ctx context.Context, dsn string, slowThreshold time.Duration) *SubData {
	pool, err := db.NewDB(ctx, dsn, slowThreshold)
	if err != nil {
		logger.Log.Fatal("failed to connect to db", zap.Error(err))
	}

	return &SubData{
		sqlDB: pool,
	}
}

type CreateArgs struct {
	ServiceName string
	MonthlyFee  int
	UserID      uuid.UUID
	StartDate   time.Time
	NMonths     int
}

func (sd *SubData) Create(ctx context.Context, args CreateArgs) (uuid.UUID, error) {
	end := args.StartDate.AddDate(0, args.NMonths, 0)

	id := uuid.New()
	query := `INSERT INTO subscriptions (id, service_name, monthly_fee, user_id, start_date, end_date)
	VALUES (@sub_id, @service_name, @monthly_fee, @user_id, @start_date, @end_date)`

	queryArgs := pgx.NamedArgs{
		"sub_id":       id,
		"service_name": args.ServiceName,
		"monthly_fee":  args.MonthlyFee,
		"user_id":      args.UserID,
		"start_date":   args.StartDate,
		"end_date":     end,
	}

	if err := sd.sqlDB.Exec(ctx, query, queryArgs); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (sd *SubData) Read(ctx context.Context, subID uuid.UUID, format string) (SubInfo, error) {
	query := `SELECT service_name, monthly_fee, user_id, start_date, end_date
	FROM subscriptions
	WHERE id = @sub_id`

	var info SubInfo
	var start, end time.Time
	queryArgs := pgx.NamedArgs{
		"sub_id": subID,
	}
	row := sd.sqlDB.QueryRow(ctx, query, queryArgs)

	scanArgs := []any{
		&info.ServiceName,
		&info.MonthlyFee,
		&info.UserID,
		&start,
		&end,
	}

	if err := row.Scan(scanArgs...); err != nil {
		return SubInfo{}, err
	}

	info.StartDate = start.Format(format)
	info.EndDate = end.Format(format)

	return info, nil
}

func (sd *SubData) Update(ctx context.Context, subID uuid.UUID, monthlyFee int, endDate time.Time) error {
	query := `UPDATE subscriptions
	SET monthly_fee = @monthly_fee, end_date = @end_date
	WHERE id = @sub_id`

	queryArgs := pgx.NamedArgs{
		"monthly_fee": monthlyFee,
		"end_date":    endDate,
		"sub_id":      subID,
	}

	return sd.sqlDB.Exec(ctx, query, queryArgs)
}

func (sd *SubData) Delete(ctx context.Context, subID uuid.UUID) error {
	query := `DELETE FROM subscriptions WHERE id = @sub_id`

	queryArgs := pgx.NamedArgs{
		"sub_id": subID,
	}

	return sd.sqlDB.Exec(ctx, query, queryArgs)
}

type PageCursor struct {
	LastStart time.Time
	LastID    uuid.UUID
	Limit     int
}

type ListArgs struct {
	ServiceName string
	UserID      uuid.UUID
	StartDate   time.Time
	EndDate     time.Time
	Format      string
	Cursor      PageCursor
}

// List - возвращает отсортированный список подписок с постраничной навигацией (курсор)
func (sd *SubData) List(ctx context.Context, args ListArgs) ([]SubInfo, *PageCursor, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT service_name, monthly_fee, user_id, start_date, end_date, id
	FROM subscriptions
	WHERE daterange(start_date, end_date, '[]') && daterange(@start_date, @end_date, '[]')`)

	if args.ServiceName != "" {
		query.WriteString(" AND service_name = @service_name")
	}

	if args.UserID != uuid.Nil {
		query.WriteString(" AND user_id = @user_id")
	}

	if !args.Cursor.LastStart.IsZero() && args.Cursor.LastID != uuid.Nil {
		query.WriteString(" AND (start_date > @last_start OR (start_date = @last_start AND id > @last_id))")
	}

	query.WriteString(" ORDER BY start_date ASC, id ASC")

	limit := args.Cursor.Limit
	query.WriteString(" LIMIT @limit")

	queryArgs := pgx.NamedArgs{
		"start_date":   args.StartDate,
		"end_date":     args.EndDate,
		"limit":        limit + 1,
		"last_start":   args.Cursor.LastStart,
		"last_id":      args.Cursor.LastID,
		"service_name": args.ServiceName,
		"user_id":      args.UserID,
	}

	rows, err := sd.sqlDB.Query(ctx, query.String(), queryArgs)
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
		if err := rows.Scan(&info.ServiceName, &info.MonthlyFee, &info.UserID, &s, &e, &id); err != nil {
			return nil, nil, err
		}

		info.StartDate = s.Format(args.Format)
		info.EndDate = e.Format(args.Format)

		res = append(res, info)

		if len(res) == limit {
			gotExtra = true
			lastStart = s
			lastID = id
			break
		}
	}

	var next *PageCursor
	if gotExtra {
		next = &PageCursor{
			LastStart: lastStart,
			LastID:    lastID,
			Limit:     limit,
		}
	}

	return res, next, nil
}

// GetSum - возвращает суммарную стоимость подписок за пересекающийся период
func (sd *SubData) GetSum(ctx context.Context, serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) (int64, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT COALESCE(SUM(COALESCE(s.monthly_fee, 0) *
		GREATEST(
			0,
			(
				(EXTRACT(YEAR FROM date_trunc('month', LEAST(s.end_date, (@end_date::date - interval '1 day')))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', LEAST(s.end_date, (@end_date::date - interval '1 day')))))
				- (EXTRACT(YEAR FROM date_trunc('month', GREATEST(s.start_date, @start_date))) * 12
				+ EXTRACT(MONTH FROM date_trunc('month', GREATEST(s.start_date, @start_date))))
				+ 1
			)
		)
	), 0) AS total_sum
	FROM subscriptions s
	WHERE daterange(start_date, end_date, '[]') && daterange(@start_date, @end_date, '[]')`)

	if serviceName != "" {
		query.WriteString(" AND service_name = @service_name")
	}

	if userID != uuid.Nil {
		query.WriteString(" AND user_id = @user_id")
	}

	queryArgs := pgx.NamedArgs{
		"start_date":   startDate,
		"end_date":     endDate,
		"service_name": serviceName,
		"user_id":      userID,
	}

	row := sd.sqlDB.QueryRow(ctx, query.String(), queryArgs)
	var res int64

	if err := row.Scan(&res); err != nil {
		return 0, err
	}

	return res, nil
}
