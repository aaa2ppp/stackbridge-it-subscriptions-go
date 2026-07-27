package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"subscriptions/internal/model"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, req model.Subscription) (model.Subscription, error)
	List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error)
	Get(ctx context.Context, id int64) (model.Subscription, error)
	Update(ctx context.Context, req model.UpdateSubscriptionRequest) (model.Subscription, error)
	Delete(ctx context.Context, id int64) error
	GetTotalCost(ctx context.Context, req model.SubscriptionFilter) (model.TotalCostResponse, error)
}

func New(svc Service) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscriptions", CreateSubscription(svc))
	mux.HandleFunc("GET /subscriptions", ListSubscriptions(svc))
	mux.HandleFunc("GET /subscriptions/{id}", GetSubscription(svc))
	mux.HandleFunc("PATCH /subscriptions/{id}", UpdateSubscription(svc))
	mux.HandleFunc("DELETE /subscriptions/{id}", DeleteSubscription(svc))
	mux.HandleFunc("GET /subscriptions/total", GetTotalCost(svc))

	return mux
}

// CreateSubscription godoc
//
//	@tags			subscriptions
//	@router			/subscriptions [post]
//	@summary		Создать подписку
//	@description	end_date - опциональное поле. Если не указано или передано как null,
//	@description	подписка считается бессрочной (внутренне используется 12-9999)
//	@accept			json
//	@produce		json
//	@param			req	body		CreateSubscriptionRequest	true	"CreateSubscriptionRequest"
//	@success		201	{object}	model.Subscription
//	@failure		400
func CreateSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "CreateSubscription")

		var req CreateSubscriptionRequest
		req.EndDate = MonthYearInfinity
		if err := h.decodeRequestBody(&req); err != nil {
			h.writeError(err)
			return
		}

		resp, err := svc.Create(h.ctx(), model.Subscription{
			ServiceName: req.ServiceName,
			UserID:      req.UserID,
			Price:       req.Price,
			StartDate:   req.StartDate,
			EndDate:     req.EndDate,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(201, resp)
	}
}

var MonthYearInfinity = model.MonthYear{Time: time.Date(9999, 12, 1, 0, 0, 0, 0, time.UTC)} // 12-9999

type CreateSubscriptionRequest struct {
	ServiceName string          `json:"service_name" validate:"required" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       int64           `json:"price" swaggertype:"integer" minimum:"0" default:"0" example:"400"`
	UserID      uuid.UUID       `json:"user_id" validate:"required" swaggertype:"string" format:"uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   model.MonthYear `json:"start_date" validate:"required" swaggertype:"string" example:"07-2025"`
	EndDate     model.MonthYear `json:"end_date" swaggertype:"string" example:"12-2025" default:"12-9999"`
}

func (req *CreateSubscriptionRequest) Validate() error {
	var errs []error

	req.ServiceName = strings.TrimSpace(req.ServiceName)
	if req.ServiceName == "" {
		errs = append(errs, errors.New("service_name cannot be empty"))
	}

	if req.Price < 0 {
		errs = append(errs, errors.New("price must be >=0"))
	}

	if req.UserID == uuid.Nil {
		errs = append(errs, errors.New("user_id is required"))
	}

	if req.StartDate.IsZero() {
		errs = append(errs, errors.New("start_date cannot be zero"))
	}

	if req.EndDate.IsZero() {
		errs = append(errs, errors.New("end_date cannot be zero"))
	}

	if req.EndDate.Before(req.StartDate.Time) {
		errs = append(errs, errors.New("end_date cannot be before start_date"))
	}

	return errors.Join(errs...)
}

const DefaultLimit = 1000

// ListSubscriptions godoc
//
//	@tags		subscriptions
//	@router		/subscriptions [get]
//	@summary	Получить список подписок
//	@param		after_id		query	integer	false	"вернуть записи следующие за after_id"	default(0)		minimum(0)
//	@param		limit			query	integer	false	"вернуть не более limit записей"		default(1000)	minimum(1)
//	@param		user_id			query	string	false	"User ID"								format(uuid)	extensions(x-example=60601fee-2bf1-4721-ae6f-7636e79a0cba)
//	@param		service_name	query	string	false	"Service Name"							minlength(1)	extensions(x-example=Yandex Plus)
//	@param		from_date		query	string	false	"From Date"								default(01-0001)
//	@param		to_date			query	string	false	"To Date"								default(12-9999)
//	@produce	json
//	@success	200	{array}	model.Subscription
//	@failure	400
func ListSubscriptions(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "ListSubscriptions")

		req, err := parseListSubscriptionsRequest(r.URL.Query())
		if err != nil {
			h.writeHTTPError(&httpError{"bad query: " + err.Error(), http.StatusBadRequest})
			return
		}

		resp, err := svc.List(h.ctx(), req)
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

func parseListSubscriptionsRequest(q url.Values) (model.ListSubscriptionsRequest, error) {
	var errs []error

	afterID := int64(0)
	if q.Has("after_id") {
		s := q.Get("after_id")
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v < 0 {
			errs = append(errs, errors.New("after_id must be integer >= 0"))
		} else {
			afterID = v
		}
	}

	limit := int64(DefaultLimit)
	if q.Has("limit") {
		s := q.Get("limit")
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v <= 0 {
			errs = append(errs, errors.New("limit must be integer > 0"))
		} else {
			limit = v
		}
	}

	filter, err := parseSubscriptionFilter(q)
	if err != nil {
		errs = append(errs, err)
	}

	return model.ListSubscriptionsRequest{
		AfterID:            afterID,
		Limit:              limit,
		SubscriptionFilter: filter,
	}, errors.Join(errs...)
}

// GetSubscription godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/{id} [get]
//	@summary	Получить подписку
//	@param		id	path	integer	true	"Subscription ID"	minimum(1)	extensions(x-example=42)
//	@produce	json
//	@success	200	{object}	model.Subscription
//	@failure	400
//	@failure	404
func GetSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "GetSubscription")

		id, err := h.getIDFromPath()
		if err != nil {
			h.writeError(err)
			return
		}

		resp, err := svc.Get(h.ctx(), id)
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

// UpdateSubscription godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/{id} [patch]
//	@summary	Обновить подписку
//	@accept		json
//	@produce	json
//	@param		id	path		integer						true	"Subscription ID"	minimum(1)	extensions(x-example=42)
//	@param		req	body		UpdateSubscriptionRequest	true	"UpdateSubscriptionRequest"
//	@success	200	{object}	model.Subscription
//	@failure	400
//	@failure	404
//	@failure	409	{object}	ConflictResponse
func UpdateSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "UpdateSubscription")

		id, err := h.getIDFromPath()
		if err != nil {
			h.writeError(err)
			return
		}

		var req UpdateSubscriptionRequest
		if err := h.decodeRequestBody(&req); err != nil {
			h.writeError(err)
			return
		}

		resp, err := svc.Update(h.ctx(), model.UpdateSubscriptionRequest{
			ID:          id,
			ServiceName: req.ServiceName,
			Price:       req.Price,
			StartDate:   req.StartDate,
			EndDate:     req.EndDate,
			Updated:     req.Updated,
		})
		if err != nil {
			if errors.Is(err, model.ErrConflict) {
				h.writeResponse(http.StatusConflict, ConflictResponse{
					Current: resp,
					Error:   err.Error(),
				})
				return
			}
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

type UpdateSubscriptionRequest struct {
	ServiceName model.Nullable[string]          `json:"service_name,omitzero" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       model.Nullable[int64]           `json:"price,omitzero" swaggertype:"integer" minimum:"0" example:"100"`
	StartDate   model.Nullable[model.MonthYear] `json:"start_date,omitzero" swaggertype:"string" example:"07-2025"`
	EndDate     model.Nullable[model.MonthYear] `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
	Updated     time.Time                       `json:"updated,omitzero" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
}

type ConflictResponse struct {
	Current model.Subscription `json:"current,omitempty"`
	Error   string             `json:"error,omitempty"`
}

func (req *UpdateSubscriptionRequest) Validate() error {
	var errs []error

	req.ServiceName.V = strings.TrimSpace(req.ServiceName.V)
	if req.ServiceName.Defined && (!req.ServiceName.Valid || req.ServiceName.V == "") {
		errs = append(errs, errors.New("service_name cannot be empty"))
	}

	if req.Price.Defined && (!req.Price.Valid || req.Price.V < 0) {
		errs = append(errs, errors.New("price must be integer >=0"))
	}

	if req.StartDate.Defined && !req.StartDate.Valid {
		errs = append(errs, errors.New("start_date cannot be null"))
	}

	if req.EndDate.Defined && !req.EndDate.Valid {
		req.EndDate.V = MonthYearInfinity
		req.EndDate.Valid = true
	}

	if req.StartDate.Defined && req.EndDate.Defined && req.EndDate.V.Before(req.StartDate.V.Time) {
		errs = append(errs, errors.New("end_date cannot be before start_date"))
	}

	return errors.Join(errs...)
}

// DeleteSubscription godoc
//
//	@tags			subscriptions
//	@router			/subscriptions/{id} [delete]
//	@summary		Удалить подписку
//	@description	Идемпотентен. Успех говорит, что подписка или была удалена или отсутствует.
//	@param			id	path	integer	true	"Subscription ID"	minimum(1)	extensions(x-example=42)
//	@produce		json
//	@success		204
//	@failure		400
func DeleteSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "DeleteSubscription")

		id, err := h.getIDFromPath()
		if err != nil {
			h.writeError(err)
			return
		}

		if err := svc.Delete(h.ctx(), id); err != nil && !errors.Is(err, model.ErrNotFound) {
			h.writeError(err)
			return
		}

		w.WriteHeader(204)
	}
}

// GetTotalCost godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/total [get]
//	@summary	Подсчет суммарной стоимости подписок
//	@produce	json
//	@param		user_id			query		string	false	"User ID"		format(uuid)	extensions(x-example=60601fee-2bf1-4721-ae6f-7636e79a0cba)
//	@param		service_name	query		string	false	"Service Name"	minlength(1)	extensions(x-example=Yandex Plus)
//	@param		from_date		query		string	false	"From Date"		extensions(x-example=07-2025)
//	@param		to_date			query		string	false	"To Date"		extensions(x-example=08-2025)
//	@success	200				{object}	model.TotalCostResponse
//	@failure	400
func GetTotalCost(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "GetTotalCost")

		req, err := parseSubscriptionFilter(r.URL.Query())
		if err != nil {
			h.writeHTTPError(&httpError{err.Error(), http.StatusBadRequest})
			return
		}

		resp, err := svc.GetTotalCost(h.ctx(), req)
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

func parseSubscriptionFilter(q url.Values) (model.SubscriptionFilter, error) {
	var req model.SubscriptionFilter
	var errs []error

	if q.Has("user_id") {
		s := q.Get("user_id")
		v, err := uuid.Parse(s)
		if err != nil {
			errs = append(errs, fmt.Errorf("user_id: %v", err))
		} else {
			req.UserID = model.Nullable[uuid.UUID]{
				Null:    sql.Null[uuid.UUID]{V: v, Valid: true},
				Defined: true,
			}
		}
	}

	if q.Has("service_name") {
		s := q.Get("service_name")
		if s == "" {
			errs = append(errs, errors.New("service_name cannot be empty"))
		} else {
			req.ServiceName = model.Nullable[string]{
				Null:    sql.Null[string]{V: s, Valid: true},
				Defined: true,
			}
		}
	}

	if q.Has("from_date") {
		s := q.Get("from_date")
		v, err := time.Parse(model.MonthYearLayout, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("from_date: %v", err))
		} else {
			req.FromDate = model.MonthYear{Time: v}
		}
	}

	if !q.Has("to_date") {
		req.ToDate = MonthYearInfinity
	} else {
		s := q.Get("to_date")
		v, err := time.Parse(model.MonthYearLayout, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("to_date: %v", err))
		} else {
			req.ToDate = model.MonthYear{Time: v}
		}
	}

	if req.ToDate.Before(req.FromDate.Time) {
		errs = append(errs, errors.New("to_date cannot be before from_date"))
	}

	return req, errors.Join(errs...)
}
