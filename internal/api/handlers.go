package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	GetTotalCost(ctx context.Context, req model.TotalCostRequest) (model.TotalCostResponse, error)
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
//	@tags		subscriptions
//	@router		/subscriptions [post]
//	@summary	Создать подписку
//	@accept		json
//	@produce	json
//	@param		req	body		CreateSubscriptionRequest	true	"CreateSubscriptionRequest"
//	@success	201	{object}	model.Subscription
//	@failure	400
//	@failure	409
func CreateSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "CreateSubscription")

		var req CreateSubscriptionRequest
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

type CreateSubscriptionRequest struct {
	ServiceName string                          `json:"service_name" validate:"required" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       int64                           `json:"price" swaggertype:"integer" minimum:"0" default:"0"`
	UserID      uuid.UUID                       `json:"user_id" validate:"required" swaggertype:"string" format:"uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   model.MonthYear                 `json:"start_date" validate:"required" swaggertype:"string" example:"07-2025"`
	EndDate     model.Nullable[model.MonthYear] `json:"end_date" swaggertype:"string" example:"12-2025"`
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

	if req.EndDate.Valid && req.EndDate.Value.Before(req.StartDate.Time) {
		errs = append(errs, errors.New("end_date cannot be before start_date"))
	}

	return errors.Join(errs...)
}

// ListSubscriptions godoc
//
//	@tags		subscriptions
//	@router		/subscriptions [get]
//	@summary	Получить список подписок
//	@param		after_id	query	integer	false	"вернуть записи следующие за after_id"	default(0)	minimum(0)
//	@param		limit		query	integer	false	"вернуть не более limit записей"		default(1)	minimum(1)
//	@produce	json
//	@success	200	{array}	model.Subscription
//	@failure	400
//	@failure	409
func ListSubscriptions(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "ListSubscriptions")
		q := h.query()

		afterID := q.Int("after_id", false, 0)
		limit := q.Int("limit", false, 1)

		if err := q.Err(); err != nil {
			h.writeError(err)
			return
		}

		if afterID < 0 {
			h.writeHTTPError(&httpError{"after_id must be >=0", http.StatusBadRequest})
		}
		if limit <= 0 {
			h.writeHTTPError(&httpError{"limit must be >0", http.StatusBadRequest})
		}

		resp, err := svc.List(h.ctx(), model.ListSubscriptionsRequest{
			AfterID: afterID,
			Limit:   limit,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
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
//	@failure	409
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
//	@failure	409	{object}	model.Subscription
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
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

type UpdateSubscriptionRequest struct {
	ServiceName model.Nullable[string]          `json:"service_name,omitzero" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       model.Nullable[int64]           `json:"price,omitzero" swaggertype:"integer" minimum:"400"`
	StartDate   model.Nullable[model.MonthYear] `json:"start_date,omitzero" swaggertype:"string" example:"07-2025"`
	EndDate     model.Nullable[model.MonthYear] `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
	Updated     time.Time                       `json:"updated,omitzero" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
}

func (req *UpdateSubscriptionRequest) Validate() error {
	var errs []error

	req.ServiceName.Value = strings.TrimSpace(req.ServiceName.Value)
	if req.ServiceName.Defined && (!req.ServiceName.Valid || req.ServiceName.Value == "") {
		errs = append(errs, errors.New("service_name cannot be empty"))
	}

	if req.Price.Defined && (!req.Price.Valid || req.Price.Value < 0) {
		errs = append(errs, errors.New("price must be integer >=0"))
	}

	if req.StartDate.Defined && !req.StartDate.Valid {
		errs = append(errs, errors.New("start_date cannot be null"))
	}

	if req.StartDate.Valid && req.EndDate.Valid && req.EndDate.Value.Before(req.StartDate.Value.Time) {
		errs = append(errs, errors.New("end_date cannot be before start_date"))
	}

	return errors.Join(errs...)
}

// DeleteSubscription godoc
//
//	@tags			subscriptions
//	@router			/subscriptions/{id} [delete]
//	@summary		Удалить подписку
//	@description	Идемпотентен. Успех говорит, что подписка или была удалена или отсутсвует.
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
//	@param		user_id			query		string	true	"User ID"		format(uuid)	extensions(x-example=60601fee-2bf1-4721-ae6f-7636e79a0cba)
//	@param		service_name	query		string	false	"Service Name"	minlength(1)	extensions(x-example=Yandex Plus)
//	@param		from_date		query		string	false	"From Date"		format(date)	extensions(x-example=2025-07-04)
//	@param		to_date			query		string	false	"To Date"		format(date)	extensions(x-example=2025-08-25)
//	@success	200				{object}	model.TotalCostResponse
//	@failure	400
func GetTotalCost(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "GetTotalCost")

		req, err := parseTotalCostRequest(r.URL.Query())
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

func parseTotalCostRequest(q url.Values) (model.TotalCostRequest, error) {
	var req model.TotalCostRequest
	var errs []error

	for range 1 {
		if !q.Has("user_id") {
			errs = append(errs, errors.New("user_id: is required"))
			break
		}
		s := q.Get("user_id")
		v, err := uuid.Parse(s)
		if err != nil {
			errs = append(errs, fmt.Errorf("user_id: %v", err))
			break
		}
		req.UserID = v
	}

	for range 1 {
		if !q.Has("service_name") {
			break
		}
		s := q.Get("service_name")
		if s == "" {
			errs = append(errs, errors.New("service_name cannot be empty"))
			break
		}
		req.ServiceName = model.Nullable[string]{
			Defined: true,
			Valid:   true,
			Value:   s,
		}
	}

	for range 1 {
		if !q.Has("from_date") {
			break
		}
		s := q.Get("from_date")
		v, err := time.Parse(model.DateLayout, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("from_date: %v", err))
			break
		}
		req.FromDate = model.Nullable[model.Date]{
			Defined: true,
			Valid:   true,
			Value:   model.Date{Time: v},
		}
	}

	for range 1 {
		if !q.Has("to_date") {
			break
		}
		s := q.Get("to_date")
		v, err := time.Parse(model.DateLayout, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("to_date: %v", err))
			break
		}
		req.ToDate = model.Nullable[model.Date]{
			Defined: true,
			Valid:   true,
			Value:   model.Date{Time: v},
		}
	}

	if req.FromDate.Valid && req.ToDate.Valid && req.ToDate.Value.Before(req.FromDate.Value.Time) {
		errs = append(errs, errors.New("to_date cannot be before from_date"))
	}

	return req, errors.Join(errs...)
}
