package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type GetOrderHistoryUseCase struct {
	orders domain.OrderRepository
}

func NewGetOrderHistoryUseCase(orders domain.OrderRepository) *GetOrderHistoryUseCase {
	return &GetOrderHistoryUseCase{orders: orders}
}

type OrderHistoryInput struct {
	CustomerID uuid.UUID
	Limit      int
	Offset     int
}

func (uc *GetOrderHistoryUseCase) Execute(ctx context.Context, input OrderHistoryInput) ([]*domain.Order, error) {
	ctx, span := tracer.Start(ctx, "GetOrderHistory")
	defer span.End()
	span.SetAttributes(attribute.String("customer_id", input.CustomerID.String()))

	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	return uc.orders.ListByCustomer(ctx, input.CustomerID, input.Limit, input.Offset)
}
