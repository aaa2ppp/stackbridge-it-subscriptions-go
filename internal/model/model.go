package model

import (
	"subscriptions/internal/model/date"

	"github.com/google/uuid"
)

type ID int64

type Subscription struct {
	ID          int64          `json:"id"`
	ServiceName string         `json:"service_name"`
	Price       int64          `json:"price"`
	UserID      uuid.UUID      `json:"user_id" swaggertype:"string" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   date.MonthYear `json:"start_date" swaggertype:"string" example:"07-2025"`
	EndDate     date.MonthYear `json:"end_date,omitzero" swaggertype:"string" example:"12-2025"`
}

type ListSubscriptionsRequest struct {
	AfterID int `json:"after_id"`
	Limit   int `json:"limit"`
}

type GetTotalCostRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	ServiceName string    `json:"service_name"`
	FromDate    date.Date `json:"from_date"  swaggertype:"string" example:"2025-07-04"`
	ToDate      date.Date `json:"to_date"  swaggertype:"string" example:"2025-12-31"`
}

type GetTotalCostResponse struct {
	TotalCost int64 `json:"total_cost"`
}
