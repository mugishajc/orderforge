package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type TransitionOrderInput struct {
	OrderID      uuid.UUID
	TargetStatus domain.OrderStatus
}

type TransitionOrderUseCase struct {
	orders domain.OrderRepository
	events domain.EventPublisher
}

func NewTransitionOrderUseCase(
	orders domain.OrderRepository,
	events domain.EventPublisher,
) *TransitionOrderUseCase {
	return &TransitionOrderUseCase{
		orders: orders,
		events: events,
	}
}

func (uc *TransitionOrderUseCase) Execute(ctx context.Context, input TransitionOrderInput) (*domain.Order, error) {
	ctx, span := tracer.Start(ctx, "TransitionOrder")
	defer span.End()
	span.SetAttributes(
		attribute.String("order_id", input.OrderID.String()),
		attribute.String("target_status", string(input.TargetStatus)),
	)

	order, err := uc.orders.GetByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	if err := order.TransitionTo(input.TargetStatus); err != nil {
		return nil, err
	}

	if err := uc.orders.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("persisting status transition: %w", err)
	}

	evt := domain.NewOrderEvent(order)
	data, _ := json.Marshal(evt)
	subject := fmt.Sprintf("orders.%s", input.TargetStatus)
	_ = uc.events.Publish(ctx, subject, data)

	return order, nil
}
