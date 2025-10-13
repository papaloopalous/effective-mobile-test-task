package handlers

import (
	"encoding/json"
	"net/http"
	"test_task/app/api/response"
	"test_task/app/internal/repo"

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
		response.WriteAPIResponse(w, http.StatusBadRequest, "invalid add request", err.Error())
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
