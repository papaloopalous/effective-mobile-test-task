package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"encoding/json"
	"task_test/internal/repo"
	"task_test/util"

	"github.com/google/uuid"
)

type stubRepo struct {
    createErr error
    readErr   error
    updateErr error
    deleteErr error
    listRes   []repo.SubInfo
    getSum    int64
    readStart string
    readEnd   string
}

func (s *stubRepo) Create(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error) {
	if s.createErr != nil {
		return uuid.Nil, s.createErr
	}
	return uuid.New(), nil
}
func (s *stubRepo) Read(subID uuid.UUID, format string) (repo.SubInfo, error) {
    if s.readErr != nil {
        return repo.SubInfo{}, s.readErr
    }
    start := s.readStart
    end := s.readEnd
    if start == "" {
        start = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(format)
    }
    if end == "" {
        end = time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Format(format)
    }
    return repo.SubInfo{ServiceName: "svc", MonthlyFee: 10, UserID: uuid.New(), StartDate: start, EndDate: end}, nil
}
func (s *stubRepo) Update(subID uuid.UUID, monthlyFee int, endDate time.Time) error {
	return s.updateErr
}
func (s *stubRepo) Delete(subID uuid.UUID) error { return s.deleteErr }
func (s *stubRepo) List(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time, format string) []repo.SubInfo {
	return s.listRes
}
func (s *stubRepo) GetSum(serviceName string, userID uuid.UUID, startDate time.Time, endDate time.Time) int64 {
	return s.getSum
}
func (s *stubRepo) Close() {}

func rec(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(bytes.NewBufferString(body)))
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func TestAddSub(t *testing.T) {
    sh := &SubHandler{Subs: &stubRepo{}}
    w := rec(sh.AddSub, `{"service_name":"a","user_id":"`+uuid.New().String()+`","monthly_fee":1,"start_date":"`+time.Date(2024,1,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`","num_months":2}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code")
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json")
	}
	if int(body["code"].(float64)) != http.StatusCreated {
		t.Fatalf("code field")
	}
	if body["message"] == "" {
		t.Fatalf("msg")
	}
	if body["data"] == nil {
		t.Fatalf("data")
	}
	w = rec(sh.AddSub, `bad`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad")
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json")
	}
	if int(body["code"].(float64)) != http.StatusBadRequest {
		t.Fatalf("code field")
	}
    w = rec(sh.AddSub, `{"service_name":"a","user_id":"`+uuid.New().String()+`","monthly_fee":1,"start_date":"bad","num_months":2}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    long := bytes.Repeat([]byte{'a'}, 256)
    w = rec(sh.AddSub, `{"service_name":"`+string(long)+`","user_id":"`+uuid.New().String()+`","monthly_fee":1,"start_date":"`+time.Date(2024,1,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`","num_months":2}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{createErr: errors.New("x")}
    w = rec(sh.AddSub, `{"service_name":"a","user_id":"`+uuid.New().String()+`","monthly_fee":1,"start_date":"`+time.Date(2024,1,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`","num_months":2}`)
    if w.Code != http.StatusInternalServerError {
        t.Fatalf("bad")
    }
}

func TestGetByID(t *testing.T) {
	sh := &SubHandler{Subs: &stubRepo{}}
	r := httptest.NewRequest(http.MethodGet, "/?sub_id="+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	sh.GetByID(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("bad")
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json")
	}
	if int(body["code"].(float64)) != http.StatusOK {
		t.Fatalf("code field")
	}
	if body["data"] == nil {
		t.Fatalf("data")
	}
	r = httptest.NewRequest(http.MethodGet, "/?sub_id=bad", nil)
	w = httptest.NewRecorder()
	sh.GetByID(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad")
	}
	sh.Subs = &stubRepo{readErr: errors.New("x")}
	r = httptest.NewRequest(http.MethodGet, "/?sub_id="+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	sh.GetByID(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("bad")
	}
}

func TestUpdateByID(t *testing.T) {
    sh := &SubHandler{Subs: &stubRepo{}}
    w := rec(sh.UpdateByID, `bad`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"bad"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{readErr: errors.New("x")}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusNotFound {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{readStart: "bad"}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusInternalServerError {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{readEnd: "bad"}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusInternalServerError {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{readStart: time.Date(2024,3,1,0,0,0,0,time.UTC).Format(util.DateFormat), readEnd: time.Date(2024,1,1,0,0,0,0,time.UTC).Format(util.DateFormat)}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusOK {
        t.Fatalf("bad")
    }
    sh.Subs = &stubRepo{updateErr: errors.New("x")}
    w = rec(sh.UpdateByID, `{"sub_id":"`+uuid.New().String()+`","monthly_fee":1,"end_date":"`+time.Date(2024,2,1,0,0,0,0,time.UTC).Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusInternalServerError {
        t.Fatalf("bad")
    }
}

func TestRemoveByID(t *testing.T) {
	sh := &SubHandler{Subs: &stubRepo{}}
	r := httptest.NewRequest(http.MethodDelete, "/?sub_id=bad", nil)
	w := httptest.NewRecorder()
	sh.RemoveByID(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad")
	}
	r = httptest.NewRequest(http.MethodDelete, "/?sub_id="+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	sh.RemoveByID(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("bad")
	}
    var body map[string]any
    if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
        t.Fatalf("json")
    }
    if int(body["code"].(float64)) != http.StatusOK {
        t.Fatalf("code field")
    }
	sh.Subs = &stubRepo{deleteErr: errors.New("x")}
	r = httptest.NewRequest(http.MethodDelete, "/?sub_id="+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	sh.RemoveByID(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("bad")
	}
}

func TestListSubsAndSum(t *testing.T) {
    uid := uuid.New()
    s := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    e := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
    sh := &SubHandler{Subs: &stubRepo{listRes: []repo.SubInfo{{ServiceName: "a", MonthlyFee: 1, UserID: uid, StartDate: s.Format(util.DateFormat), EndDate: e.Format(util.DateFormat)}}, getSum: 123}}
    body := `{"service_name":"a","user_id":"` + uid.String() + `","start_date":"` + s.Format(util.DateFormat) + `","end_date":"` + e.Format(util.DateFormat) + `"}`
    w := rec(sh.ListSubs, body)
    if w.Code != http.StatusOK {
        t.Fatalf("bad")
    }
    var lresp struct {
        Code    int                    `json:"code"`
        Message string                 `json:"message"`
        Data    []map[string]any       `json:"data"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &lresp); err != nil {
        t.Fatalf("json")
    }
    if len(lresp.Data) != 1 {
        t.Fatalf("len")
    }
    if lresp.Data[0]["service_name"] != "a" || int(lresp.Data[0]["monthly_fee"].(float64)) != 1 {
        t.Fatalf("values")
    }
    if lresp.Data[0]["start_date"] != s.Format(util.DateFormat) || lresp.Data[0]["end_date"] != e.Format(util.DateFormat) {
        t.Fatalf("dates")
    }
    w = rec(sh.SumSubs, body)
    if w.Code != http.StatusOK {
        t.Fatalf("bad")
    }
    var sresp struct {
        Code    int     `json:"code"`
        Message string  `json:"message"`
        Data    float64 `json:"data"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &sresp); err != nil {
        t.Fatalf("json")
    }
    if int64(sresp.Data) != 123 {
        t.Fatalf("sum")
    }
    w = rec(sh.SumSubs, body)
    if w.Code != http.StatusOK {
        t.Fatalf("bad")
    }
    w = rec(sh.ListSubs, `bad`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    long := bytes.Repeat([]byte{'a'}, 256)
    w = rec(sh.ListSubs, `{"service_name":"`+string(long)+`","user_id":"","start_date":"`+s.Format(util.DateFormat)+`","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.ListSubs, `{"service_name":"a","user_id":"bad","start_date":"`+s.Format(util.DateFormat)+`","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.ListSubs, `{"service_name":"a","user_id":"","start_date":"bad","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Body.Len() != 0 {
        t.Fatalf("bad")
    }
    w = rec(sh.ListSubs, `{"service_name":"a","user_id":"","start_date":"`+s.Format(util.DateFormat)+`","end_date":"bad"}`)
    if w.Body.Len() != 0 {
        t.Fatalf("bad")
    }
    w = rec(sh.SumSubs, `bad`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.SumSubs, `{"service_name":"`+string(long)+`","user_id":"","start_date":"`+s.Format(util.DateFormat)+`","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.SumSubs, `{"service_name":"a","user_id":"bad","start_date":"`+s.Format(util.DateFormat)+`","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("bad")
    }
    w = rec(sh.SumSubs, `{"service_name":"a","user_id":"","start_date":"bad","end_date":"`+e.Format(util.DateFormat)+`"}`)
    if w.Body.Len() != 0 {
        t.Fatalf("bad")
    }
    w = rec(sh.SumSubs, `{"service_name":"a","user_id":"","start_date":"`+s.Format(util.DateFormat)+`","end_date":"bad"}`)
    if w.Body.Len() != 0 {
        t.Fatalf("bad")
    }
}
