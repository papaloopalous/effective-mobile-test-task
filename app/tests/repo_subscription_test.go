package tests

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	dbpkg "task_test/internal/db"
	repoPkg "task_test/internal/repo"
	"task_test/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	vals    []any
	scanErr error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}

	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(r.vals[i]))
	}

	return nil
}

type fakeRows struct {
	data    [][]any
	idx     int
	scanErr error
	err     error
}

func (r *fakeRows) Close() {
}

func (r *fakeRows) Err() error {
	return r.err
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	if r.idx < len(r.data) {
		r.idx++
		return true
	}

	return false
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}

	if r.idx == 0 || r.idx-1 >= len(r.data) {
		return errors.New("no row")
	}

	row := r.data[r.idx-1]
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}

	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	if r.idx == 0 || r.idx-1 >= len(r.data) {
		return nil, errors.New("no row")
	}

	return r.data[r.idx-1], nil
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}

type fakeDB struct {
	lastQuery string
	lastArgs  []any
	execErr   error
	queryErr  error
	row       pgx.Row
	rows      pgx.Rows
}

// QueryRow - фиксирует последний запрос/аргументы и возвращает подготовленную строку
func (f *fakeDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	f.lastQuery = query
	f.lastArgs = args
	return f.row
}

// Query - фиксирует последний запрос/аргументы и возвращает подготовленный набор строк/ошибку
func (f *fakeDB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	f.lastQuery = query
	f.lastArgs = args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.rows, nil
}

func (f *fakeDB) Ping(ctx context.Context) error {
	return nil
}

// Exec - фиксирует последний запрос/аргументы и возвращает ошибку (если задана)
func (f *fakeDB) Exec(ctx context.Context, query string, args ...any) error {
	f.lastQuery = query
	f.lastArgs = args
	return f.execErr
}

func (f *fakeDB) Close() {
}

func setSubDataDB(sd *repoPkg.SubData, db dbpkg.DB) {
	rv := reflect.ValueOf(sd).Elem().FieldByName("sqlDB")
	ptr := unsafe.Pointer(rv.UnsafeAddr())
	reflect.NewAt(rv.Type(), ptr).Elem().Set(reflect.ValueOf(db))
}

// TestRepo_Create_OK_Error - успешное создание и ошибка Exec
func TestRepo_Create_OK_Error(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{}

	setSubDataDB(sd, f)

	uid := uuid.New()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	args := repoPkg.CreateArgs{
		ServiceName: "s",
		MonthlyFee:  10,
		UserID:      uid,
		StartDate:   start,
		NMonths:     2,
	}
	_, err := sd.Create(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}

	args = repoPkg.CreateArgs{
		ServiceName: "s",
		MonthlyFee:  10,
		UserID:      uid,
		StartDate:   start,
		NMonths:     1,
	}
	f.execErr = errors.New("x")
	_, err = sd.Create(context.Background(), args)
	if err == nil {
		t.Fatal("expected err")
	}
}

// TestRepo_Read_OK_Error - успешное чтение и ошибка Scan
func TestRepo_Read_OK_Error(t *testing.T) {
	sd := &repoPkg.SubData{}

	uid := uuid.New()
	s := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	f := &fakeDB{row: &fakeRow{vals: []any{"svc", 5, uid, s, e}}}

	setSubDataDB(sd, f)

	out, err := sd.Read(context.Background(), uuid.New(), util.DateFormat)
	if err != nil {
		t.Fatal(err)
	}

	if out.ServiceName != "svc" || out.MonthlyFee != 5 || out.UserID != uid || out.StartDate != "01-2025" || out.EndDate != "02-2025" {
		t.Fatal("bad read")
	}

	f.row = &fakeRow{scanErr: errors.New("x")}
	_, err = sd.Read(context.Background(), uuid.New(), util.DateFormat)
	if err == nil {
		t.Fatal("expected err")
	}
}

// TestRepo_Update_Delete - успешные и ошибочные случаи обновления/удаления
func TestRepo_Update_Delete(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{}

	setSubDataDB(sd, f)

	err := sd.Update(context.Background(), uuid.New(), 10, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	f.execErr = errors.New("x")
	if err = sd.Update(context.Background(), uuid.New(), 10, time.Now()); err == nil {
		t.Fatal("upd err")
	}

	f.execErr = nil
	if err = sd.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatal(err)
	}

	f.execErr = errors.New("x")
	if err = sd.Delete(context.Background(), uuid.New()); err == nil {
		t.Fatal("del err")
	}
}

// TestRepo_List_QueryErr_ScanErr_OK_NoNext_WithNext - разные сценарии листинга и курсора
func TestRepo_List_QueryErr_ScanErr_OK_NoNext_WithNext(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{}

	setSubDataDB(sd, f)

	args := repoPkg.ListArgs{
		ServiceName: "",
		UserID:      uuid.Nil,
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		Format:      util.DateFormat,
		Cursor: repoPkg.PageCursor{
			Limit: 2,
		},
	}

	f.queryErr = errors.New("x")
	_, _, err := sd.List(context.Background(), args)
	if err == nil {
		t.Fatal("query err")
	}

	rows := &fakeRows{data: [][]any{{"s", 1, uuid.New(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), uuid.New()}}, scanErr: errors.New("x")}
	f.queryErr = nil
	f.rows = rows
	_, _, err = sd.List(context.Background(), args)
	if err == nil {
		t.Fatal("scan err")
	}

	rows.scanErr = nil
	rows.data = [][]any{
		{"s1", 1, uuid.New(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), uuid.New()},
	}
	rows.idx = 0
	f.rows = rows
	res, next, err := sd.List(context.Background(), args)
	if err != nil || len(res) != 1 || next != nil {
		t.Fatal("list no next")
	}

	if res[0].StartDate != "01-2025" || res[0].EndDate != "02-2025" {
		t.Fatal("fmt")
	}

	idLast := uuid.New()
	rows.data = [][]any{
		{"s1", 1, uuid.New(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), uuid.New()},
		{"s2", 2, uuid.New(), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), idLast},
		{"s3", 3, uuid.New(), time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), uuid.New()},
	}
	rows.idx = 0
	res, next, err = sd.List(context.Background(), args)
	if err != nil || len(res) != 2 || next == nil || next.LastID != idLast {
		t.Fatal("list next")
	}
}

// TestRepo_List_WithFiltersAndCursor - проверка фильтров и курсора в SQL
func TestRepo_List_WithFiltersAndCursor(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{}

	setSubDataDB(sd, f)

	args := repoPkg.ListArgs{
		ServiceName: "svc",
		UserID:      uuid.New(),
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		Format:      util.DateFormat,
		Cursor: repoPkg.PageCursor{
			LastStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			LastID:    uuid.New(),
			Limit:     1,
		},
	}

	rows := &fakeRows{data: [][]any{{"s", 1, uuid.New(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), uuid.New()}}}
	f.rows = rows
	_, _, err := sd.List(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
}

// TestRepo_GetSum_OK_Err - успешная агрегация и ошибка Scan
func TestRepo_GetSum_OK_Err(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{row: &fakeRow{vals: []any{int64(123)}}}

	setSubDataDB(sd, f)

	sum, err := sd.GetSum(context.Background(), "", uuid.Nil, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || sum != 123 {
		t.Fatal("sum ok")
	}

	f.row = &fakeRow{scanErr: errors.New("x")}
	_, err = sd.GetSum(context.Background(), "", uuid.Nil, time.Now(), time.Now())
	if err == nil {
		t.Fatal("sum err")
	}
}

// TestRepo_GetSum_WithFilters - проверка применения фильтров и аргументов
func TestRepo_GetSum_WithFilters(t *testing.T) {
	sd := &repoPkg.SubData{}
	f := &fakeDB{row: &fakeRow{vals: []any{int64(77)}}}

	setSubDataDB(sd, f)

	uid := uuid.New()
	s := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	sum, err := sd.GetSum(context.Background(), "svc", uid, s, e)
	if err != nil || sum != 77 {
		t.Fatalf("sum %v err %v", sum, err)
	}

	if !strings.Contains(f.lastQuery, "service_name") || !strings.Contains(f.lastQuery, "user_id") {
		t.Fatal("filters not in query")
	}
}

// TestRepo_Close - закрытие репозитория
func TestRepo_Close(t *testing.T) {
	sd := &repoPkg.SubData{}
	setSubDataDB(sd, &fakeDB{})
	sd.Close()
}
