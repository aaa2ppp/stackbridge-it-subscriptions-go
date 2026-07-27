package model

import (
	"time"

	"github.com/google/uuid"
)

type ID int64

type Subscription struct {
	ID          int64               `json:"id"`
	ServiceName string              `json:"service_name"`
	Price       int64               `json:"price"`
	UserID      uuid.UUID           `json:"user_id" swaggertype:"string" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   MonthYear           `json:"start_date" swaggertype:"string" example:"07-2025"`
	EndDate     MonthYear           `json:"end_date" swaggertype:"string" example:"12-2025"`
	Created     time.Time           `json:"created" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
	Updated     time.Time           `json:"updated" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
	Deleted     Nullable[time.Time] `json:"deleted,omitzero" swaggertype:"string" format:"date-time" example:"2025-12-01T14:38:00.000Z"`
}

type ListSubscriptionsRequest struct {
	SubscriptionFilter
	AfterID int64 `json:"after_id"`
	Limit   int64 `json:"limit"`
}

type UpdateSubscriptionRequest struct {
	ID          int64               `json:"id"`
	ServiceName Nullable[string]    `json:"service_name,omitzero"`
	Price       Nullable[int64]     `json:"price,omitzero"`
	StartDate   Nullable[MonthYear] `json:"start_date,omitzero" swaggertype:"string" example:"07-2025"`
	EndDate     Nullable[MonthYear] `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
	Updated     time.Time           `json:"updated"` // для оптимистичной блокировки
}

type SubscriptionFilter struct {
	UserID      Nullable[uuid.UUID] `json:"user_id,omitzero"`
	ServiceName Nullable[string]    `json:"service_name,omitzero"`
	FromDate    MonthYear           `json:"from_date" swaggertype:"string" example:"07-2025"`
	ToDate      MonthYear           `json:"to_date" swaggertype:"string" example:"12-2025"`
}

type TotalCostResponse struct {
	TotalCost int64 `json:"total_cost"`
}
