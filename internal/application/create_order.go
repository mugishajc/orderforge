package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("orderforge")

type CreateOrderInput struct {
	CustomerID uuid.UUID
	Items      []CreateOrderItemInput
}

type CreateOrderItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type CreateOrderUseCase struct {
	orders   domain.OrderRepository
	products domain.ProductRepository
	cache    domain.ProductCache
	events   domain.EventPublisher
}

func NewCreateOrderUseCase(
	orders domain.OrderRepository,
	products domain.ProductRepository,
	cache domain.ProductCache,
	events domain.EventPublisher,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orders:   orders,
		products: products,
		cache:    cache,
		events:   events,
	}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	ctx, span := tracer.Start(ctx, "CreateOrder")
	defer span.End()

	span.SetAttributes(attribute.String("customer_id", input.CustomerID.String()))

	// Build order items with prices from the product catalog.
	// Reserve inventory for each item.
	var items []domain.OrderItem
	reserved := make([]struct {
		product *domain.Product
		qty     int
	}, 0, len(input.Items))

	for _, in := range input.Items {
		product, err := uc.lookupProduct(ctx, in.ProductID)
		if err != nil {
			uc.rollbackReservations(ctx, reserved)
			return nil, err
		}

		if err := product.Reserve(in.Quantity); err != nil {
			uc.rollbackReservations(ctx, reserved)
			return nil, fmt.Errorf("product %s: %w", product.SKU, err)
		}
		reserved = append(reserved, struct {
			product *domain.Product
			qty     int
		}{product, in.Quantity})

		items = append(items, domain.OrderItem{
			ProductID: in.ProductID,
			Quantity:  in.Quantity,
			UnitPrice: product.Price,
		})
	}

	order, err := domain.NewOrder(input.CustomerID, items)
	if err != nil {
		uc.rollbackReservations(ctx, reserved)
		return nil, err
	}

	// Persist inventory changes.
	for _, r := range reserved {
		if err := uc.products.Update(ctx, r.product); err != nil {
			return nil, fmt.Errorf("updating product stock: %w", err)
		}
		_ = uc.cache.Delete(ctx, r.product.ID)
	}

	if err := uc.orders.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("saving order: %w", err)
	}

	// Publish order.created event.
	evt := domain.NewOrderEvent(order)
	data, _ := json.Marshal(evt)
	_ = uc.events.Publish(ctx, "orders.created", data)

	span.SetAttributes(attribute.String("order_id", order.ID.String()))
	return order, nil
}

func (uc *CreateOrderUseCase) lookupProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	// Try cache first.
	if p, err := uc.cache.Get(ctx, id); err == nil && p != nil {
		return p, nil
	}
	p, err := uc.products.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = uc.cache.Set(ctx, p)
	return p, nil
}

func (uc *CreateOrderUseCase) rollbackReservations(ctx context.Context, reserved []struct {
	product *domain.Product
	qty     int
}) {
	for _, r := range reserved {
		r.product.Release(r.qty)
		_ = uc.products.Update(ctx, r.product)
	}
}
