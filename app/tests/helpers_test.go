package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
)

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newReq(method, path, body string) (*httptest.ResponseRecorder, *http.Request) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return rr, req
}

func decodeResp(rr *httptest.ResponseRecorder) (apiResp, error) {
	var ar apiResp
	err := json.Unmarshal(rr.Body.Bytes(), &ar)
	return ar, err
}
