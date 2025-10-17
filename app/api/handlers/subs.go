package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"task_test/api/response"
	"task_test/internal/repo"
	"task_test/util"

	"time"

	"github.com/google/uuid"
)

type SubHandler struct {
	Subs repo.SubRepo
}

type addReq struct {
	ServiceName string    `json:"service_name"`
	UserID      uuid.UUID `json:"user_id"`
	MonthlyFee  int       `json:"monthly_fee"`
	StartDate   string    `json:"start_date"`
	NMonths     int       `json:"num_months"`
}

func (sh *SubHandler) AddSub(w http.ResponseWriter, r *http.Request) {

	var req addReq

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidAddReq, err.Error())
		return
	}

	start, err := time.Parse(util.DateFormat, req.StartDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseStartDate, err.Error())
		return
	}

	switch {

	case req.NMonths <= 0:
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogNonPosNMonths, nil)
		return

	case req.MonthlyFee < 0:
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogNegMonthlyFee, nil)
		return

	case len(req.ServiceName) > 255:
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogLongServiceName, nil)
		return

	}

	id, err := sh.Subs.Create(req.ServiceName, req.MonthlyFee, req.UserID, start, req.NMonths)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogAddSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusCreated, util.SuccessLogAddSub, id)
}

func (sh *SubHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidSubID, err.Error())
		return
	}

	sub, err := sh.Subs.Read(subID, util.DateFormat)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogGetSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SussessLogGetSub, sub)
}

type updateReq struct {
	SubID      uuid.UUID `json:"sub_id"`
	MonthlyFee int       `json:"monthly_fee"`
	EndDate    string    `json:"end_date"`
}

func (sh *SubHandler) UpdateByID(w http.ResponseWriter, r *http.Request) {
	var req updateReq

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidUpdateReq, err.Error())
		return
	}

	end, err := time.Parse(util.DateFormat, req.EndDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseEndDate, err.Error())
		return
	}

	info, err := sh.Subs.Read(req.SubID, util.DateFormat)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusNotFound, util.ErrLogSubNotFound, err.Error())
		return
	}

	var s time.Time
	s, err = time.Parse(util.DateFormat, info.StartDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogParseStartDate, err.Error())
		return
	}

	switch {

	case req.MonthlyFee < 0:
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogNegMonthlyFee, nil)
		return

	case s.Compare(end) == 1:
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLodEndBeforeStartDate, map[string]string{"start": s.Format(util.DateFormat), "end": end.Format(util.DateFormat)})
		return

	}

	err = sh.Subs.Update(req.SubID, req.MonthlyFee, end)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogUpdateSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogUpdateSub, nil)
}

func (sh *SubHandler) RemoveByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidSubID, err.Error())
		return
	}

	err = sh.Subs.Delete(subID)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogDeleteSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogDeleteSub, nil)
}

type cursorIn struct {
	LastStart string `json:"last_start,omitempty"`
	LastID    string `json:"last_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type listReq struct {
	ServiceName string   `json:"service_name"`
	UserID      string   `json:"user_id"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	Cursor      cursorIn `json:"cursor,omitempty"`
}

type cursorOut struct {
	LastStart string `json:"last_start"`
	LastID    string `json:"last_id"`
	Limit     int    `json:"limit"`
}

type listResp struct {
	Subscriptions []repo.SubInfo `json:"subscriptions"`
	Cursor        *cursorOut     `json:"cursor,omitempty"`
}

type listReqValidated struct {
	serviceName string
	userID      uuid.UUID
	startDate   time.Time
	endDate     time.Time
	lastStart   time.Time
	lastID      uuid.UUID
	limit       int
}

func decodeAndValidateListReq(w http.ResponseWriter, r *http.Request) (listReqValidated, error) {
	var req listReq

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidListReq, err.Error())
		return listReqValidated{}, err
	}

	if len(req.ServiceName) > 255 {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogLongServiceName, nil)
		return listReqValidated{}, errors.New("service name too long")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil && req.UserID != "" {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidUserID, err.Error())
		return listReqValidated{}, err
	}

	if req.UserID == "" {
		userID = uuid.Nil
	}

	var s, e time.Time
	s, err = time.Parse(util.DateFormat, req.StartDate)
	if err != nil {
		return listReqValidated{}, err
	}

	e, err = time.Parse(util.DateFormat, req.EndDate)
	if err != nil {
		return listReqValidated{}, err
	}

	var lastID uuid.UUID
	var lastStart time.Time
	limit := 50
	if req.Cursor.LastStart != "" {
		lastStart, err = time.Parse(util.DateFormat, req.Cursor.LastStart)
		if err != nil {
			response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseCursorLastStart, err.Error())
			return listReqValidated{}, err
		}
	}

	if req.Cursor.LastID != "" {
		lastID, err = uuid.Parse(req.Cursor.LastID)
		if err != nil {
			response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidCursorLastID, err.Error())
			return listReqValidated{}, err
		}
	}

	if req.Cursor.Limit > 0 {
		limit = req.Cursor.Limit
	}

	reqV := listReqValidated{
		serviceName: req.ServiceName,
		userID:      userID,
		startDate:   s,
		endDate:     e,
		lastStart:   lastStart,
		lastID:      lastID,
		limit:       limit,
	}

	return reqV, nil
}

func (sh *SubHandler) ListSubs(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	cur := repo.PageCursor{
		LastStart: req.lastStart,
		LastID:    req.lastID,
		Limit:     req.limit,
	}

	subs, nextCur, err := sh.Subs.List(req.serviceName, req.userID, req.startDate, req.endDate, util.DateFormat, cur)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogListSub, err.Error())
		return
	}

	var next *cursorOut
	if nextCur != nil && nextCur.LastID != uuid.Nil && !nextCur.LastStart.IsZero() {
		next = &cursorOut{
			LastStart: nextCur.LastStart.Format(util.DateFormat),
			LastID:    nextCur.LastID.String(),
			Limit:     nextCur.Limit,
		}
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogListSubs, listResp{
		Subscriptions: subs,
		Cursor:        next,
	})
}

func (sh *SubHandler) SumSubs(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	sum, err := sh.Subs.GetSum(req.serviceName, req.userID, req.startDate, req.endDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogSumSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogSumSubs, sum)
}
