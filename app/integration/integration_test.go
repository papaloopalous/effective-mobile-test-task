//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"task_test/api/handlers"
	"task_test/internal/repo"
	"task_test/util"
)

func getDSN(t *testing.T) string {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set; skipping integration tests")
	}
	return dsn
}

func ensureSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS subscriptions (
        id UUID PRIMARY KEY,
        service_name VARCHAR(255) NOT NULL,
        monthly_fee INT NOT NULL,
        user_id UUID NOT NULL,
        start_date DATE NOT NULL,
        end_date DATE NOT NULL
    );`)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM subscriptions;`)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func newPool(t *testing.T, dsn string) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestRepoIntegration_CRUD_List_Sum(t *testing.T) {
	dsn := getDSN(t)
	pool := newPool(t, dsn)
	ensureSchema(t, pool)

	subRepo := repo.NewSubRepo(dsn, 200*time.Millisecond)
	t.Cleanup(subRepo.Close)

	uid1 := uuid.New()
	start1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id1, err := subRepo.Create("a", 10, uid1, start1, 3)
	if err != nil || id1 == uuid.Nil {
		t.Fatalf("create1")
	}

	got1, err := subRepo.Read(id1, util.DateFormat)
	if err != nil {
		t.Fatalf("read1")
	}
	if got1.ServiceName != "a" || got1.MonthlyFee != 10 || got1.UserID != uid1 {
		t.Fatalf("read1 values")
	}
	if got1.StartDate != start1.Format(util.DateFormat) || got1.EndDate != start1.AddDate(0, 3, 0).Format(util.DateFormat) {
		t.Fatalf("read1 dates")
	}

	err = subRepo.Update(id1, 99, start1.AddDate(0, 4, 0))
	if err != nil {
		t.Fatalf("update1")
	}
	gotU, err := subRepo.Read(id1, util.DateFormat)
	if err != nil || gotU.MonthlyFee != 99 || gotU.EndDate != start1.AddDate(0, 4, 0).Format(util.DateFormat) {
		t.Fatalf("updated values")
	}

	uid2 := uuid.New()
	start2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	id2, err := subRepo.Create("a", 20, uid2, start2, 1)
	if err != nil || id2 == uuid.Nil {
		t.Fatalf("create2")
	}
	_, err = subRepo.Create("b", 30, uid2, start2, 2)
	if err != nil {
		t.Fatalf("create3")
	}

	list := subRepo.List("a", uuid.Nil, start1, start1.AddDate(0, 4, 0), util.DateFormat)
	if len(list) != 2 {
		t.Fatalf("list len")
	}
	if list[0].ServiceName != "a" || list[1].ServiceName != "a" {
		t.Fatalf("list values")
	}

	sumAll := subRepo.GetSum("", uuid.Nil, start1, start1.AddDate(0, 4, 0))
	if sumAll <= 0 {
		t.Fatalf("sum all: %d", sumAll)
	}

	sumA := subRepo.GetSum("a", uuid.Nil, start1, start1.AddDate(0, 4, 0))
	if sumA <= 0 || sumA > sumAll {
		t.Fatalf("sum a invalid: all=%d a=%d", sumAll, sumA)
	}

	err = subRepo.Delete(id2)
	if err != nil {
		t.Fatalf("delete2")
	}
}

func rec(h http.HandlerFunc, method, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, io.NopCloser(bytes.NewBufferString(body)))
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func TestHandlersIntegration_WithRealDB(t *testing.T) {
	_ = os.MkdirAll("logs", 0o755)

	dsn := getDSN(t)
	pool := newPool(t, dsn)
	ensureSchema(t, pool)

	subRepo := repo.NewSubRepo(dsn, 200*time.Millisecond)
	t.Cleanup(subRepo.Close)
	h := &handlers.SubHandler{Subs: subRepo}

	uid := uuid.New()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	w := rec(h.AddSub, http.MethodPost, "/addSub", `{"service_name":"int","user_id":"`+uid.String()+`","monthly_fee":10,"start_date":"`+start.Format(util.DateFormat)+`","num_months":2}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("add code")
	}
	var addResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &addResp); err != nil || addResp.Data == "" {
		t.Fatalf("add parse")
	}

	w = rec(h.GetByID, http.MethodGet, "/getSub?sub_id="+addResp.Data, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get code")
	}
	var getResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("get parse")
	}
	if getResp.Data["service_name"] != "int" {
		t.Fatalf("get values")
	}

	end := start.AddDate(0, 3, 0)
	w = rec(h.UpdateByID, http.MethodPut, "/updateSub", `{"sub_id":"`+addResp.Data+`","monthly_fee":99,"end_date":"`+end.Format(util.DateFormat)+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("upd code")
	}

	body := `{"service_name":"int","user_id":"` + uid.String() + `","start_date":"` + start.Format(util.DateFormat) + `","end_date":"` + end.Format(util.DateFormat) + `"}`
	w = rec(h.ListSubs, http.MethodGet, "/listSubs", body)
	if w.Code != http.StatusOK {
		t.Fatalf("list code")
	}
	var listResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil || len(listResp.Data) == 0 {
		t.Fatalf("list parse")
	}

	w = rec(h.SumSubs, http.MethodGet, "/totalSubs", body)
	if w.Code != http.StatusOK {
		t.Fatalf("sum code")
	}
	var sumResp struct {
		Data float64 `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sumResp); err != nil || int64(sumResp.Data) <= 0 {
		t.Fatalf("sum parse")
	}

	w = rec(h.RemoveByID, http.MethodDelete, "/deleteSub?sub_id="+addResp.Data, "")
	if w.Code != http.StatusOK {
		t.Fatalf("del code")
	}
}
