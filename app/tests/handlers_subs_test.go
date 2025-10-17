package tests

import (
	"errors"
	"net/http"
	"task_test/api/handlers"
	repoPkg "task_test/internal/repo"
	"task_test/util"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAddSub_DecodeError(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	rr, req := newReq(http.MethodPost, "/addSub", `{"unknown":1}`)

	h.AddSub(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rr.Code)
	}

	ar, _ := decodeResp(rr)
	if ar.Message != util.ErrLogInvalidAddReq {
		t.Fatalf("msg %s", ar.Message)
	}
}

func TestAddSub_ParseStartError(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	body := `{"service_name":"s","user_id":"` + uuid.New().String() + `","monthly_fee":10,"start_date":"13-2025","num_months":1}`
	rr, req := newReq(http.MethodPost, "/addSub", body)

	h.AddSub(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rr.Code)
	}

	ar, _ := decodeResp(rr)
	if ar.Message != util.ErrLogParseStartDate {
		t.Fatalf("msg %s", ar.Message)
	}
}

func TestAddSub_ValidateErrors(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	uid := uuid.New().String()
	b1 := `{"service_name":"s","user_id":"` + uid + `","monthly_fee":10,"start_date":"01-2025","num_months":0}`
	rr, req := newReq(http.MethodPost, "/addSub", b1)

	h.AddSub(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogNonPosNMonths {
		t.Fatal("nmonths")
	}

	b2 := `{"service_name":"s","user_id":"` + uid + `","monthly_fee":-1,"start_date":"01-2025","num_months":1}`
	rr, req = newReq(http.MethodPost, "/addSub", b2)

	h.AddSub(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogNegMonthlyFee {
		t.Fatal("fee")
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}

	b3 := `{"service_name":"` + string(long) + `","user_id":"` + uid + `","monthly_fee":10,"start_date":"01-2025","num_months":1}`
	rr, req = newReq(http.MethodPost, "/addSub", b3)

	h.AddSub(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogLongServiceName {
		t.Fatal("name")
	}
}

func TestAddSub_CreateErrorAndOK(t *testing.T) {
	uid := uuid.New()
	hErr := &handlers.SubHandler{Subs: &mockSubRepo{createFn: func(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error) {
		return uuid.Nil, errors.New("x")
	}}}
	body := `{"service_name":"s","user_id":"` + uid.String() + `","monthly_fee":10,"start_date":"01-2025","num_months":1}`
	rr, req := newReq(http.MethodPost, "/addSub", body)

	hErr.AddSub(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogAddSub {
		t.Fatal("add err")
	}

	newID := uuid.New()
	hOK := &handlers.SubHandler{Subs: &mockSubRepo{createFn: func(serviceName string, monthlyFee int, userID uuid.UUID, startDate time.Time, nMonths int) (uuid.UUID, error) {
		return newID, nil
	}}}
	rr, req = newReq(http.MethodPost, "/addSub", body)

	hOK.AddSub(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatal(rr.Code)
	}

	ar, _ := decodeResp(rr)
	if ar.Message != util.SuccessLogAddSub {
		t.Fatal("add ok")
	}
}

func TestGetByID_ParseAndRepoErrorsAndOK(t *testing.T) {
	hBad := &handlers.SubHandler{Subs: &mockSubRepo{}}
	rr, req := newReq(http.MethodGet, "/getSub?sub_id=bad", "")

	hBad.GetByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidSubID {
		t.Fatal("id parse")
	}

	id := uuid.New()
	hErr := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(subID uuid.UUID, format string) (repoPkg.SubInfo, error) {
		return repoPkg.SubInfo{}, errors.New("x")
	}}}
	rr, req = newReq(http.MethodGet, "/getSub?sub_id="+id.String(), "")

	hErr.GetByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogGetSub {
		t.Fatal("get err")
	}

	info := repoPkg.SubInfo{ServiceName: "s", MonthlyFee: 10, UserID: uuid.New(), StartDate: "01-2025", EndDate: "02-2025"}
	hOK := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(subID uuid.UUID, format string) (repoPkg.SubInfo, error) { return info, nil }}}
	rr, req = newReq(http.MethodGet, "/getSub?sub_id="+id.String(), "")

	hOK.GetByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SussessLogGetSub {
		t.Fatal("get ok")
	}
}

func TestUpdateByID_DecodeAndParseErrors(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	rr, req := newReq(http.MethodPut, "/updateSub", `{"unknown":1}`)

	h.UpdateByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidUpdateReq {
		t.Fatal("decode")
	}

	id := uuid.New()
	body := `{"sub_id":"` + id.String() + `","monthly_fee":10,"end_date":"13-2025"}`
	rr, req = newReq(http.MethodPut, "/updateSub", body)

	h.UpdateByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogParseEndDate {
		t.Fatal("end parse")
	}
}

func TestUpdateByID_NotFound_ParseStart_NegFee_EndBeforeStart_UpdateErr_OK(t *testing.T) {
	id := uuid.New()
	hNotFound := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(subID uuid.UUID, format string) (repoPkg.SubInfo, error) {
		return repoPkg.SubInfo{}, errors.New("nf")
	}}}
	rr, req := newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":10,"end_date":"02-2025"}`)

	hNotFound.UpdateByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogSubNotFound {
		t.Fatal("nf")
	}

	badInfo := repoPkg.SubInfo{StartDate: "bad"}
	hParse := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(uuid.UUID, string) (repoPkg.SubInfo, error) { return badInfo, nil }}}
	rr, req = newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":10,"end_date":"02-2025"}`)

	hParse.UpdateByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogParseStartDate {
		t.Fatal("start parse")
	}

	info := repoPkg.SubInfo{StartDate: "01-2025"}
	hNeg := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(uuid.UUID, string) (repoPkg.SubInfo, error) { return info, nil }}}
	rr, req = newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":-1,"end_date":"02-2025"}`)

	hNeg.UpdateByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogNegMonthlyFee {
		t.Fatal("neg")
	}

	info2 := repoPkg.SubInfo{StartDate: "03-2025"}
	hOrder := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(uuid.UUID, string) (repoPkg.SubInfo, error) { return info2, nil }}}
	rr, req = newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":0,"end_date":"02-2025"}`)

	hOrder.UpdateByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLodEndBeforeStartDate {
		t.Fatal("order")
	}

	info3 := repoPkg.SubInfo{StartDate: "01-2025"}
	hUpdErr := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(uuid.UUID, string) (repoPkg.SubInfo, error) { return info3, nil }, updateFn: func(uuid.UUID, int, time.Time) error { return errors.New("x") }}}
	rr, req = newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":1,"end_date":"02-2025"}`)

	hUpdErr.UpdateByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogUpdateSub {
		t.Fatal("upd err")
	}

	hOK := &handlers.SubHandler{Subs: &mockSubRepo{readFn: func(uuid.UUID, string) (repoPkg.SubInfo, error) { return info3, nil }, updateFn: func(uuid.UUID, int, time.Time) error { return nil }}}
	rr, req = newReq(http.MethodPut, "/updateSub", `{"sub_id":"`+id.String()+`","monthly_fee":1,"end_date":"02-2025"}`)

	hOK.UpdateByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SuccessLogUpdateSub {
		t.Fatal("upd ok")
	}
}

func TestRemoveByID_Parse_DeleteErr_OK(t *testing.T) {
	hBad := &handlers.SubHandler{Subs: &mockSubRepo{}}

	rr, req := newReq(http.MethodDelete, "/deleteSub?sub_id=bad", "")

	hBad.RemoveByID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidSubID {
		t.Fatal("parse")
	}

	id := uuid.New()
	hErr := &handlers.SubHandler{Subs: &mockSubRepo{deleteFn: func(uuid.UUID) error { return errors.New("x") }}}
	rr, req = newReq(http.MethodDelete, "/deleteSub?sub_id="+id.String(), "")

	hErr.RemoveByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogDeleteSub {
		t.Fatal("del err")
	}

	hOK := &handlers.SubHandler{Subs: &mockSubRepo{deleteFn: func(uuid.UUID) error { return nil }}}
	rr, req = newReq(http.MethodDelete, "/deleteSub?sub_id="+id.String(), "")

	hOK.RemoveByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SuccessLogDeleteSub {
		t.Fatal("del ok")
	}
}

func TestListSubs_DecodeErrors(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	rr, req := newReq(http.MethodGet, "/listSubs", `{"unknown":1}`)

	h.ListSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidListReq {
		t.Fatal("inv")
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"`+string(long)+`","user_id":"","start_date":"01-2025","end_date":"02-2025"}`)

	h.ListSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogLongServiceName {
		t.Fatal("long")
	}

	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"a","user_id":"bad","start_date":"01-2025","end_date":"02-2025"}`)

	h.ListSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidUserID {
		t.Fatal("uid")
	}

	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"a","user_id":"","start_date":"01-2025","end_date":"02-2025","cursor":{"last_start":"bad"}}`)

	h.ListSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogParseCursorLastStart {
		t.Fatal("lstart")
	}

	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"a","user_id":"","start_date":"01-2025","end_date":"02-2025","cursor":{"last_id":"bad"}}`)

	h.ListSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidCursorLastID {
		t.Fatal("lid")
	}

	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"a","user_id":"","start_date":"bad","end_date":"02-2025"}`)
	h.ListSubs(rr, req)

	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"a","user_id":"","start_date":"01-2025","end_date":"bad"}`)
	h.ListSubs(rr, req)
}

func TestListSubs_ListErr_NoCursor_WithCursor(t *testing.T) {
	hErr := &handlers.SubHandler{Subs: &mockSubRepo{listFn: func(string, uuid.UUID, time.Time, time.Time, string, repoPkg.PageCursor) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error) {
		return nil, nil, errors.New("x")
	}}}

	rr, req := newReq(http.MethodGet, "/listSubs", `{"service_name":"","user_id":"","start_date":"01-2025","end_date":"02-2025"}`)

	hErr.ListSubs(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogListSub {
		t.Fatal("list err")
	}

	subs := []repoPkg.SubInfo{{ServiceName: "s", MonthlyFee: 1, UserID: uuid.New(), StartDate: "01-2025", EndDate: "02-2025"}}
	hNoCur := &handlers.SubHandler{Subs: &mockSubRepo{listFn: func(string, uuid.UUID, time.Time, time.Time, string, repoPkg.PageCursor) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error) {
		return subs, &repoPkg.PageCursor{}, nil
	}}}
	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"","user_id":"","start_date":"01-2025","end_date":"02-2025"}`)

	hNoCur.ListSubs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SuccessLogListSubs {
		t.Fatal("list ok no cur")
	}

	next := &repoPkg.PageCursor{LastStart: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), LastID: uuid.New(), Limit: 1}
	hCur := &handlers.SubHandler{Subs: &mockSubRepo{listFn: func(string, uuid.UUID, time.Time, time.Time, string, repoPkg.PageCursor) ([]repoPkg.SubInfo, *repoPkg.PageCursor, error) {
		return subs, next, nil
	}}}
	rr, req = newReq(http.MethodGet, "/listSubs", `{"service_name":"","user_id":"","start_date":"01-2025","end_date":"02-2025","cursor":{"limit":1}}`)

	hCur.ListSubs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SuccessLogListSubs {
		t.Fatal("list ok with cur")
	}
}

func TestSumSubs_GetErr_OK(t *testing.T) {
	hErr := &handlers.SubHandler{Subs: &mockSubRepo{sumFn: func(string, uuid.UUID, time.Time, time.Time) (int64, error) { return 0, errors.New("x") }}}

	rr, req := newReq(http.MethodGet, "/totalSubs", `{"service_name":"","user_id":"","start_date":"01-2025","end_date":"02-2025"}`)

	hErr.SumSubs(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogSumSub {
		t.Fatal("sum err")
	}

	hOK := &handlers.SubHandler{Subs: &mockSubRepo{sumFn: func(string, uuid.UUID, time.Time, time.Time) (int64, error) { return 123, nil }}}
	rr, req = newReq(http.MethodGet, "/totalSubs", `{"service_name":"","user_id":"","start_date":"01-2025","end_date":"02-2025"}`)

	hOK.SumSubs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.SuccessLogSumSubs {
		t.Fatal("sum ok")
	}
}

func TestSumSubs_DecodeErrors(t *testing.T) {
	h := &handlers.SubHandler{Subs: &mockSubRepo{}}

	rr, req := newReq(http.MethodGet, "/totalSubs", `{"unknown":1}`)

	h.SumSubs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	if ar, _ := decodeResp(rr); ar.Message != util.ErrLogInvalidListReq {
		t.Fatal("inv")
	}
}
