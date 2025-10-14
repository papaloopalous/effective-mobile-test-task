package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"task_test/api/response"
	"task_test/internal/repo"

	"time"

	"github.com/google/uuid"
)

type SubHandler struct {
	Subs repo.SubRepo
}

// TODO: посмотреть где лучше парсить дату, в хендлере или в репо

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
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid add request", err.Error())
		return
	}

	if len(req.ServiceName) > 255 {
		response.WriteAPIResponse(w, http.StatusBadRequest, "service name is too long, 255 is allowed", nil)
		return
	}

	id, err := sh.Subs.Create(req.ServiceName, req.MonthlyFee, req.UserID, req.StartDate, req.NMonths)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to add a subscription", err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusCreated, "subscription added successfully", id)
}

func (sh *SubHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid subscription id", err.Error())
		return
	}

	sub, err := sh.Subs.Read(subID)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to get a subscription", err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, "subscription retrieved successfully", sub)
}

type updateReq struct {
	SubID      uuid.UUID `json:"sub_id"`
	MonthlyFee int       `json:"monthly_fee"`
	EndDate    string    `json:"end_date"`
}

func (sh *SubHandler) UpdateByID(w http.ResponseWriter, r *http.Request) {
	var req updateReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid update request", err.Error())
		return
	}

	info, err := sh.Subs.Read(req.SubID)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusNotFound, "subscription not found", err.Error())
		return
	}

	var s, e time.Time
	s, err = time.Parse("01-2006", info.StartDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to parse start date", err.Error())
		return
	}
	e, err = time.Parse("01-2006", req.EndDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to parse end date", err.Error())
		return
	}

	if s.Compare(e) == 1 {
		response.WriteAPIResponse(w, http.StatusBadRequest, "end date could not be before start date", map[string]string{"start": s.Format("01-2006"), "end": e.Format("01-2006")})
		return
	}

	err = sh.Subs.Update(req.SubID, req.MonthlyFee, req.EndDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to update a subscription", err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, "subscription updated successfully", nil)
}

func (sh *SubHandler) RemoveByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid subscription id", err.Error())
		return
	}

	err = sh.Subs.Delete(subID)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, "failed to delete a subscription", err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, "subscription deleted successfully", nil)
}

type listReq struct {
	ServiceName string `json:"service_name"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

func decodeAndValidateListReq(w http.ResponseWriter, r *http.Request) (listReq, uuid.UUID, error) {
	var req listReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid list request", err.Error())
		return listReq{}, uuid.Nil, err
	}

	if len(req.ServiceName) > 255 {
		response.WriteAPIResponse(w, http.StatusBadRequest, "service name is too long, 255 is allowed", nil)
		return listReq{}, uuid.Nil, errors.New("service name too long")
	}

	var userID uuid.UUID
	userID, err := uuid.Parse(req.UserID)
	if err != nil && req.UserID != "" {
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid user id", err.Error())
		return listReq{}, uuid.Nil, err
	}

	if req.UserID == "" {
		userID = uuid.Nil
	}

	return req, userID, nil
}

func (sh *SubHandler) ListSubs(w http.ResponseWriter, r *http.Request) {
	req, userID, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	subs := sh.Subs.List(req.ServiceName, userID, req.StartDate, req.EndDate)

	response.WriteAPIResponse(w, http.StatusOK, "subscriptions listed successfully", subs)
}

func (sh *SubHandler) SumSubs(w http.ResponseWriter, r *http.Request) {
	req, userID, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	sum := sh.Subs.GetSum(req.ServiceName, userID, req.StartDate, req.EndDate)

	response.WriteAPIResponse(w, http.StatusOK, "total sum listed successfully", sum)
}
