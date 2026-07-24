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
	EndDate     Nullable[MonthYear] `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
	Created     time.Time           `json:"created" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
	Updated     time.Time           `json:"updated" swaggertype:"string" format:"date-time" example:"2025-07-01T14:38:00.000Z"`
}

type ListSubscriptionsRequest struct {
	AfterID int `json:"after_id"`
	Limit   int `json:"limit"`
}

type UpdateSubscriptionRequest struct {
	ID          int64               `json:"id"`
	ServiceName Nullable[string]    `json:"service_name,omitzero"`
	Price       Nullable[int64]     `json:"price,omitzero"`
	StartDate   Nullable[MonthYear] `json:"start_date,omitzero" swaggertype:"string" example:"07-2025"`
	EndDate     Nullable[MonthYear] `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
	Updated     time.Time           `json:"updated"` // для оптимистичной блокировки
}

type TotalCostRequest struct {
	UserID      uuid.UUID        `json:"user_id"`
	ServiceName Nullable[string] `json:"service_name"`
	FromDate    Nullable[Date]   `json:"from_date"  swaggertype:"string" example:"2025-07-04"`
	ToDate      Nullable[Date]   `json:"to_date"  swaggertype:"string" example:"2025-12-31"`
}

type TotalCostResponse struct {
	TotalCost int64 `json:"total_cost"`
}
