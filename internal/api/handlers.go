package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"subscriptions/internal/lib/nullable"
	"subscriptions/internal/model"
	"subscriptions/internal/model/date"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, req model.Subscription) (model.Subscription, error)
	List(ctx context.Context, req model.ListSubscriptionsRequest) ([]model.Subscription, error)
	Get(ctx context.Context, id int64) (model.Subscription, error)
	Update(ctx context.Context, req model.Subscription) (model.Subscription, error)
	Delete(ctx context.Context, id int64) error
	GetTotalCost(ctx context.Context, req model.GetTotalCostRequest) (model.GetTotalCostResponse, error)
}

func New(svc Service) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscriptions", CreateSubscription(svc))
	mux.HandleFunc("GET /subscriptions", ListSubscriptions(svc))
	mux.HandleFunc("GET /subscriptions/{id}", GetSubscription(svc))
	mux.HandleFunc("PUT /subscriptions/{id}", UpdateSubscription(svc))
	mux.HandleFunc("DELETE /subscriptions/{id}", DeleteSubscription(svc))
	mux.HandleFunc("GET subscriptions/total", GetTotalCost(svc))

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
			ServiceName: req.ServiceName.Value,
			UserID:      req.UserID.Value,
			Price:       req.Price.Value,
			StartDate:   req.StartDate.Value,
			EndDate:     req.EndDate.Value,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(201, resp)
	}
}

type CreateSubscriptionRequest struct {
	ServiceName nullable.Nullable[string]         `json:"service_name" validate:"required" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       nullable.Nullable[int64]          `json:"price" validate:"required" swaggertype:"integer" minimum:"400"`
	UserID      nullable.Nullable[uuid.UUID]      `json:"user_id" validate:"required" swaggertype:"string" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   nullable.Nullable[date.MonthYear] `json:"start_date" validate:"required" swaggertype:"string" example:"07-2025"`
	EndDate     nullable.Nullable[date.MonthYear] `json:"end_date" swaggertype:"string" example:"12-2025"`
}

func (req *CreateSubscriptionRequest) Validate() error {
	var errs []error

	req.ServiceName.Value = strings.TrimSpace(req.ServiceName.Value)
	if !req.ServiceName.Valid {
		errs = append(errs, errors.New("service_name is required"))
	} else if req.ServiceName.Value == "" {
		errs = append(errs, errors.New("service_name cannot be empty"))
	}

	if !req.Price.Valid {
		errs = append(errs, errors.New("price is required"))
	} else if req.Price.Value < 0 {
		errs = append(errs, errors.New("price must be >=0"))
	}

	if !req.UserID.Valid {
		errs = append(errs, errors.New("user_id is required"))
	}

	if !req.StartDate.Valid {
		errs = append(errs, errors.New("start_date is required"))
	}

	if req.EndDate.Valid && req.EndDate.Value.Before(req.StartDate.Value.Time) {
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
//	@param		id	path	integer	true	"Subscription ID"	minimum(1)	example(42)
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

// TODO: возможно нужно переделать UpdateSubscription на PATCH

// UpdateSubscription godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/{id} [put]
//	@summary	Обновить подписку
//	@accept		json
//	@produce	json
//	@param		id	path		integer						true	"Subscription ID"	minimum(1)	example(42)
//	@param		req	body		UpdateSubscriptionRequest	true	"UpdateSubscriptionRequest"
//	@success	200	{object}	model.Subscription
//	@failure	400
//	@failure	404
//	@failure	409
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

		resp, err := svc.Update(h.ctx(), model.Subscription{
			ID:          id,
			ServiceName: req.ServiceName.Value,
			Price:       req.Price.Value,
			StartDate:   req.StartDate.Value,
			EndDate:     req.EndDate.Value,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(200, resp)
	}
}

type UpdateSubscriptionRequest struct {
	ServiceName nullable.Nullable[string]         `json:"service_name" validate:"required" swaggertype:"string" minlength:"1" example:"Yandex Plus"`
	Price       nullable.Nullable[int64]          `json:"price" validate:"required" swaggertype:"integer" minimum:"400"`
	StartDate   nullable.Nullable[date.MonthYear] `json:"start_date" validate:"required" swaggertype:"string" example:"07-2025"`
	EndDate     nullable.Nullable[date.MonthYear] `json:"end_date" swaggertype:"string" example:"12-2025"`
}

func (req *UpdateSubscriptionRequest) Validate() error {
	var errs []error

	req.ServiceName.Value = strings.TrimSpace(req.ServiceName.Value)
	if !req.ServiceName.Valid {
		errs = append(errs, errors.New("service_name is required"))
	} else if req.ServiceName.Value == "" {
		errs = append(errs, errors.New("service_name cannot be empty"))
	}

	if !req.Price.Valid {
		errs = append(errs, errors.New("price is required"))
	} else if req.Price.Value < 0 {
		errs = append(errs, errors.New("price must be >=0"))
	}

	if !req.StartDate.Valid {
		errs = append(errs, errors.New("start_date is required"))
	}

	if req.EndDate.Valid && req.EndDate.Value.Before(req.StartDate.Value.Time) {
		errs = append(errs, errors.New("end_date cannot be before start_date"))
	}

	return errors.Join(errs...)
}

// DeleteSubscription godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/{id} [delete]
//	@summary	Удалить подписку
//	@param		id	path	integer	true	"Subscription ID"	minimum(1)	example(42)
//	@produce	json
//	@success	200
//	@failure	400
func DeleteSubscription(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "DeleteSubscription")

		id, err := h.getIDFromPath()
		if err != nil {
			h.writeError(err)
			return
		}

		if err := svc.Delete(h.ctx(), id); err != nil {
			h.writeError(err)
			return
		}
	}
}

// GetTotalCost godoc
//
//	@tags		subscriptions
//	@router		/subscriptions/total [get]
//	@summary	Подсчет суммарной стоимости подписок
//	@produce	json
//	@success	200	{object}	model.GetTotalCostResponse
//	@failure	400
func GetTotalCost(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "GetTotalCost")
		h.writeHTTPError(&httpError{"not implemented", http.StatusNotImplemented})
	}
}
