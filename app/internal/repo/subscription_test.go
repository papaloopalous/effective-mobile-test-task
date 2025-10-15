package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockRow struct {
	vals []any
	err  error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.vals[i].(string)
		case *int:
			*d = r.vals[i].(int)
		case *int64:
			*d = r.vals[i].(int64)
		case *uuid.UUID:
			*d = r.vals[i].(uuid.UUID)
		case *time.Time:
			*d = r.vals[i].(time.Time)
		default:
			return errors.New("bad dest type")
		}
	}
	return nil
}

type mockRows struct {
	data      [][]any
	idx       int
	err       error
	scanErrAt int
}

func (r *mockRows) Close()                                       {}
func (r *mockRows) Err() error                                   { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Next() bool                                   { return r.idx < len(r.data) }
func (r *mockRows) Scan(dest ...any) error {
	i := r.idx
	r.idx++
	if r.scanErrAt == i {
		return errors.New("scan error")
	}
	row := r.data[i]
	for j := range dest {
		switch d := dest[j].(type) {
		case *string:
			*d = row[j].(string)
		case *int:
			*d = row[j].(int)
		case *uuid.UUID:
			*d = row[j].(uuid.UUID)
		case *time.Time:
			*d = row[j].(time.Time)
		default:
			return errors.New("bad dest type")
		}
	}
	return nil
}
func (r *mockRows) Values() ([]any, error) { return nil, nil }
func (r *mockRows) RawValues() [][]byte    { return nil }
func (r *mockRows) Conn() *pgx.Conn        { return nil }

type mockDB struct {
	lastQuery string
	lastArgs  []any
	execErr   error
	queryErr  error
	row       pgx.Row
	rows      pgx.Rows
	closed    bool
}

func (m *mockDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	m.lastQuery = query
	m.lastArgs = args
	return m.row
}
func (m *mockDB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	m.lastQuery = query
	m.lastArgs = args
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.rows, nil
}
func (m *mockDB) Ping(ctx context.Context) error { return nil }
func (m *mockDB) Exec(ctx context.Context, query string, args ...any) error {
	m.lastQuery = query
	m.lastArgs = args
	return m.execErr
}
func (m *mockDB) Close() { m.closed = true }

func TestCreate_OK(t *testing.T) {
	m := &mockDB{}
	sd := &SubData{sqlDB: m}
	uid := uuid.New()
	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	id, err := sd.Create("svc", 123, uid, start, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == uuid.Nil {
		t.Fatalf("id is nil")
	}
	if len(m.lastArgs) != 6 {
		t.Fatalf("bad args count")
	}
	if m.lastArgs[1] != "svc" || m.lastArgs[2] != 123 {
		t.Fatalf("bad args")
	}
	if m.lastArgs[3] != uid {
		t.Fatalf("bad uid")
	}
	if !m.lastArgs[4].(time.Time).Equal(start) {
		t.Fatalf("bad start")
	}
	if !m.lastArgs[5].(time.Time).Equal(start.AddDate(0, 2, 0)) {
		t.Fatalf("bad end")
	}
}

func TestCreate_Err(t *testing.T) {
	m := &mockDB{execErr: errors.New("x")}
	sd := &SubData{sqlDB: m}
	id, err := sd.Create("a", 1, uuid.New(), time.Now(), 1)
	if err == nil || id != uuid.Nil {
		t.Fatalf("expected error")
	}
}

func TestRead_OK(t *testing.T) {
	svc := "svc"
	fee := 77
	uid := uuid.New()
	s := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	e := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	m := &mockDB{row: &mockRow{vals: []any{svc, fee, uid, s, e}}}
	sd := &SubData{sqlDB: m}
	info, err := sd.Read(uuid.New(), "2006-01-02 15:04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ServiceName != svc || info.MonthlyFee != fee || info.UserID != uid {
		t.Fatalf("bad values")
	}
	if info.StartDate != s.Format("2006-01-02 15:04") || info.EndDate != e.Format("2006-01-02 15:04") {
		t.Fatalf("bad date format")
	}
}

func TestRead_Err(t *testing.T) {
	m := &mockDB{row: &mockRow{err: errors.New("x")}}
	sd := &SubData{sqlDB: m}
	_, err := sd.Read(uuid.New(), time.RFC3339)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdate_OK_And_Err(t *testing.T) {
	m := &mockDB{}
	sd := &SubData{sqlDB: m}
	err := sd.Update(uuid.New(), 55, time.Now())
	if err != nil {
		t.Fatalf("unexpected error")
	}
	m.execErr = errors.New("x")
	if sd.Update(uuid.New(), 55, time.Now()) == nil {
		t.Fatalf("expected error")
	}
}

func TestDelete_OK_And_Err(t *testing.T) {
	m := &mockDB{}
	sd := &SubData{sqlDB: m}
	if err := sd.Delete(uuid.New()); err != nil {
		t.Fatalf("unexpected error")
	}
	m.execErr = errors.New("x")
	if sd.Delete(uuid.New()) == nil {
		t.Fatalf("expected error")
	}
}

func TestList_OK_ScanErr_QueryErr(t *testing.T) {
	uid := uuid.New()
	s1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e1 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	s2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	e2 := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	rows := &mockRows{data: [][]any{{"a", 10, uid, s1, e1}, {"b", 20, uid, s2, e2}}, scanErrAt: 0}
	m := &mockDB{rows: rows}
	sd := &SubData{sqlDB: m}
	res := sd.List("", uuid.Nil, s1, e2, "2006-01-02")
	if len(res) != 1 {
		t.Fatalf("expected 1 row")
	}
	if res[0].ServiceName != "b" || res[0].MonthlyFee != 20 || res[0].UserID != uid {
		t.Fatalf("bad values")
	}
	if res[0].StartDate != s2.Format("2006-01-02") || res[0].EndDate != e2.Format("2006-01-02") {
		t.Fatalf("bad date format")
	}
	m.queryErr = errors.New("x")
	if len(sd.List("x", uid, s1, e2, time.RFC3339)) != 0 {
		t.Fatalf("expected empty")
	}
}

func TestGetSum_OK_And_ScanErr(t *testing.T) {
	m := &mockDB{row: &mockRow{vals: []any{int64(123)}}}
	sd := &SubData{sqlDB: m}
	v := sd.GetSum("", uuid.Nil, time.Now(), time.Now())
	if v != 123 {
		t.Fatalf("bad sum")
	}
	m.row = &mockRow{err: errors.New("x")}
	v = sd.GetSum("x", uuid.New(), time.Now(), time.Now())
	if v != 0 {
		t.Fatalf("expected 0 on error")
	}
}

func TestClose(t *testing.T) {
	m := &mockDB{}
	sd := &SubData{sqlDB: m}
	sd.Close()
	if !m.closed {
		t.Fatalf("not closed")
	}
}
