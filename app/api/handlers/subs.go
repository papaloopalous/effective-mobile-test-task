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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidAddReq, err.Error())
		return
	}

	start, err := time.Parse(util.DateFormat, req.StartDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseStartDate, err.Error())
		return
	}

	if len(req.ServiceName) > 255 {
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	var s, e time.Time
	s, err = time.Parse(util.DateFormat, info.StartDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogParseStartDate, err.Error())
	}

	e, err = time.Parse(util.DateFormat, info.EndDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogParseEndDate, err.Error())
	}

	if s.Compare(e) == 1 {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLodEndBeforeStartDate, map[string]string{"start": s.Format(util.DateFormat), "end": e.Format(util.DateFormat)})
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

type listReq struct {
	ServiceName string `json:"service_name"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type listReqValidated struct {
	serviceName string
	userID      string
	startDate   time.Time
	endDate     time.Time
}

func decodeAndValidateListReq(w http.ResponseWriter, r *http.Request) (listReqValidated, uuid.UUID, error) {
	var req listReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidListReq, err.Error())
		return listReqValidated{}, uuid.Nil, err
	}

	if len(req.ServiceName) > 255 {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogLongServiceName, nil)
		return listReqValidated{}, uuid.Nil, errors.New("service name too long")
	}

	var userID uuid.UUID
	userID, err := uuid.Parse(req.UserID)
	if err != nil && req.UserID != "" {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidUserID, err.Error())
		return listReqValidated{}, uuid.Nil, err
	}

	if req.UserID == "" {
		userID = uuid.Nil
	}

	var s, e time.Time
	s, err = time.Parse(util.DateFormat, req.StartDate)
	if err != nil {
		return listReqValidated{}, uuid.Nil, err
	}

	e, err = time.Parse(util.DateFormat, req.EndDate)
	if err != nil {
		return listReqValidated{}, uuid.Nil, err
	}

	reqV := listReqValidated{
		serviceName: req.ServiceName,
		userID:      req.UserID,
		startDate:   s,
		endDate:     e,
	}

	return reqV, userID, nil
}

func (sh *SubHandler) ListSubs(w http.ResponseWriter, r *http.Request) {
	req, userID, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	subs := sh.Subs.List(req.serviceName, userID, req.startDate, req.endDate, util.DateFormat)

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogListSubs, subs)
}

func (sh *SubHandler) SumSubs(w http.ResponseWriter, r *http.Request) {
	req, userID, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	sum := sh.Subs.GetSum(req.serviceName, userID, req.startDate, req.endDate)

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogSumSubs, sum)
}
