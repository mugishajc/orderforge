package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrderEvent is published on every order state transition.
type OrderEvent struct {
	EventID    uuid.UUID   `json:"event_id"`
	OrderID    uuid.UUID   `json:"order_id"`
	CustomerID uuid.UUID   `json:"customer_id"`
	Status     OrderStatus `json:"status"`
	Total      int64       `json:"total"`
	OccurredAt time.Time   `json:"occurred_at"`
}

func NewOrderEvent(order *Order) OrderEvent {
	return OrderEvent{
		EventID:    uuid.New(),
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		Total:      order.Total,
		OccurredAt: time.Now().UTC(),
	}
}
