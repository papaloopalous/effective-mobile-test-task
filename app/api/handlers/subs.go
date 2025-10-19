package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"task_test/api/response"
	"task_test/internal/repo"
	"task_test/util"

	"github.com/google/uuid"
)

type SubHandler struct {
	Subs    repo.SubRepo
	Timeout time.Duration
}

// AddReq - тело запроса для создания подписки
type AddReq struct {
	ServiceName string    `json:"service_name" example:"Netflix"`                         // название сервиса
	UserID      uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"` // ID пользователя (UUID)
	MonthlyFee  int       `json:"monthly_fee" example:"499"`                              // ежемесячная стоимость
	StartDate   string    `json:"start_date" example:"01-2025"`                           // дата начала (MM-YYYY)
	NMonths     int       `json:"num_months" example:"12"`                                // длительность подписки в месяцах
}

// AddSub - создаёт новую подписку
// @Summary Создать подписку
// @Description Создаёт новую подписку для указанного пользователя и сервиса. Возвращает id созданной подписки.
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param request body handlers.AddReq true "Тело запроса для создания подписки"
// @Success 201 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /addSub [post]
func (sh *SubHandler) AddSub(w http.ResponseWriter, r *http.Request) {
	var req AddReq

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

	args := repo.CreateArgs{
		ServiceName: req.ServiceName,
		MonthlyFee:  req.MonthlyFee,
		UserID:      req.UserID,
		StartDate:   start,
		NMonths:     req.NMonths,
	}

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	id, err := sh.Subs.Create(ctx, args)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogAddSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusCreated, util.SuccessLogAddSub, id)
}

// GetByID - возвращает подписку по её ID
// @Summary Получить подписку по ID
// @Description Возвращает информацию о подписке по UUID.
// @Tags Subscriptions
// @Produce json
// @Param sub_id query string true "ID подписки (UUID)"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /getSub [get]
func (sh *SubHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidSubID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	sub, err := sh.Subs.Read(ctx, subID, util.DateFormat)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogGetSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SussessLogGetSub, sub)
}

// UpdateReq - тело запроса для обновления подписки
type UpdateReq struct {
	SubID      uuid.UUID `json:"sub_id" example:"2f1e4e2a-1a7d-4e6b-a222-3cb3f92f0a11"` // ID подписки (UUID)
	MonthlyFee int       `json:"monthly_fee" example:"599"`                             // новая ежемесячная стоимость
	EndDate    string    `json:"end_date" example:"01-2025"`                            // новая дата окончания (MM-YYYY)
}

// UpdateByID - обновляет подписку по её ID
// @Summary Обновить подписку
// @Description Обновляет ежемесячную стоимость и дату окончания подписки.
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param request body handlers.UpdateReq true "Тело запроса для обновления подписки"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /updateSub [put]
func (sh *SubHandler) UpdateByID(w http.ResponseWriter, r *http.Request) {
	var req UpdateReq

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

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	info, err := sh.Subs.Read(ctx, req.SubID, util.DateFormat)
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

	if err := sh.Subs.Update(ctx, req.SubID, req.MonthlyFee, end); err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogUpdateSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogUpdateSub, nil)
}

// RemoveByID - удаляет подписку по её ID
// @Summary Удалить подписку
// @Description Удаляет подписку по UUID.
// @Tags Subscriptions
// @Produce json
// @Param sub_id query string true "ID подписки (UUID)"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /deleteSub [delete]
func (sh *SubHandler) RemoveByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sub_id")
	subID, err := uuid.Parse(id)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogInvalidSubID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	if err := sh.Subs.Delete(ctx, subID); err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogDeleteSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogDeleteSub, nil)
}

// CursorIn - входной курсор пагинации
type CursorIn struct {
	LastStart string `json:"last_start,omitempty" example:"01-2025"`                           // последняя дата (MM-YYYY) (необязательно)
	LastID    string `json:"last_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"` // последний ID (UUID) (необязательно)
	Limit     int    `json:"limit,omitempty" example:"50"`                                     // размер страницы (по умолчанию 50) (необязательно)
}

// ListReq - фильтр для выборки/суммирования подписок
type ListReq struct {
	ServiceName string   `json:"service_name" example:"Netflix"` // фильтр по названию сервиса (необязательно)
	UserID      string   `json:"user_id" example:""`             // фильтр по пользователю (UUID, необязательно)
	StartDate   string   `json:"start_date" example:"01-2025"`   // начальная дата периода (необязательно)
	EndDate     string   `json:"end_date" example:""`            // конечная дата периода (необязательно)
	Cursor      CursorIn `json:"cursor"`                         // курсор пагинации (необязательно)
}

// CursorOut - курсор для следующей страницы в ответе
type CursorOut struct {
	LastStart string `json:"last_start"`
	LastID    string `json:"last_id"`
	Limit     int    `json:"limit"`
}

// ListResp - список подписок и курсор следующей страницы
type ListResp struct {
	Subscriptions []repo.SubInfo `json:"subscriptions"`
	Cursor        *CursorOut     `json:"cursor,omitempty"`
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
	var req ListReq

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
	if req.StartDate != "" {
		s, err = time.Parse(util.DateFormat, req.StartDate)
		if err != nil {
			response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseStartDate, err.Error())
			return listReqValidated{}, err
		}
	}

	if req.EndDate == "" {
		e = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	} else {
		e, err = time.Parse(util.DateFormat, req.EndDate)
		if err != nil {
			response.WriteAPIResponse(w, http.StatusBadRequest, util.ErrLogParseEndDate, err.Error())
			return listReqValidated{}, err
		}
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

// ListSubs - возвращает список подписок с пагинацией
// Важно: эндпоинт принимает JSON в теле (для примера используется метод POST).
// @Summary Список подписок
// @Description Возвращает список подписок, отфильтрованных по сервису, пользователю и диапазону дат. Поддерживает курсорную пагинацию. В случае отсутствия фильтров возвращает все подписки.
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param request body handlers.ListReq true "Фильтры и курсор пагинации"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /listSubs [post]
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

	args := repo.ListArgs{
		ServiceName: req.serviceName,
		UserID:      req.userID,
		StartDate:   req.startDate,
		EndDate:     req.endDate,
		Format:      util.DateFormat,
		Cursor:      cur,
	}

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	subs, nextCur, err := sh.Subs.List(ctx, args)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogListSub, err.Error())
		return
	}

	var next *CursorOut
	if nextCur != nil && nextCur.LastID != uuid.Nil && !nextCur.LastStart.IsZero() {
		next = &CursorOut{
			LastStart: nextCur.LastStart.Format(util.DateFormat),
			LastID:    nextCur.LastID.String(),
			Limit:     nextCur.Limit,
		}
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogListSubs, ListResp{
		Subscriptions: subs,
		Cursor:        next,
	})
}

// SumSubs - возвращает общую сумму по подходящим подпискам
// Важно: эндпоинт принимает JSON в теле (для примера используется метод POST).
// @Summary Общая сумма по подпискам
// @Description Рассчитывает суммарную стоимость подписок по заданным фильтрам за пересекающийся период. В случае отсутствия фильтров рассчитывает сумму по всем подпискам.
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param request body handlers.ListReq true "Фильтры для расчёта суммы"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /totalSubs [post]
func (sh *SubHandler) SumSubs(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAndValidateListReq(w, r)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sh.Timeout)
	defer cancel()

	sum, err := sh.Subs.GetSum(ctx, req.serviceName, req.userID, req.startDate, req.endDate)
	if err != nil {
		response.WriteAPIResponse(w, http.StatusInternalServerError, util.ErrLogSumSub, err.Error())
		return
	}

	response.WriteAPIResponse(w, http.StatusOK, util.SuccessLogSumSubs, sum)
}
