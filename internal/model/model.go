package model

import (
	"subscriptions/internal/model/monthyear"

	"github.com/google/uuid"
)

type Subscriptions struct {
	ID          int64               `json:"id,omitempty"`
	ServiceName string              `json:"service_name,omitempty"`
	Price       int64               `json:"price,omitempty"`
	UserID      uuid.UUID           `json:"user_id,omitempty" swaggertype:"string" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   monthyear.MonthYear `json:"start_date,omitzero" swaggertype:"string" example:"07-2025"`
}
