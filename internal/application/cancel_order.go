package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type CancelOrderUseCase struct {
	orders   domain.OrderRepository
	products domain.ProductRepository
	cache    domain.ProductCache
	events   domain.EventPublisher
}

func NewCancelOrderUseCase(
	orders domain.OrderRepository,
	products domain.ProductRepository,
	cache domain.ProductCache,
	events domain.EventPublisher,
) *CancelOrderUseCase {
	return &CancelOrderUseCase{
		orders:   orders,
		products: products,
		cache:    cache,
		events:   events,
	}
}

func (uc *CancelOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "CancelOrder")
	defer span.End()
	span.SetAttributes(attribute.String("order_id", orderID.String()))

	order, err := uc.orders.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if err := order.TransitionTo(domain.OrderStatusCancelled); err != nil {
		return err
	}

	// Release reserved inventory.
	for _, item := range order.Items {
		product, err := uc.products.GetByID(ctx, item.ProductID)
		if err != nil {
			continue // best effort: log and proceed
		}
		product.Release(item.Quantity)
		if err := uc.products.Update(ctx, product); err != nil {
			continue
		}
		_ = uc.cache.Delete(ctx, product.ID)
	}

	if err := uc.orders.Update(ctx, order); err != nil {
		return fmt.Errorf("updating cancelled order: %w", err)
	}

	evt := domain.NewOrderEvent(order)
	data, _ := json.Marshal(evt)
	_ = uc.events.Publish(ctx, "orders.cancelled", data)

	return nil
}
