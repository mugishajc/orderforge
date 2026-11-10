package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type ApplyDiscountInput struct {
	OrderID    uuid.UUID
	CouponCode string
}

type ApplyDiscountUseCase struct {
	orders  domain.OrderRepository
	coupons domain.CouponRepository
	events  domain.EventPublisher
}

func NewApplyDiscountUseCase(
	orders domain.OrderRepository,
	coupons domain.CouponRepository,
	events domain.EventPublisher,
) *ApplyDiscountUseCase {
	return &ApplyDiscountUseCase{
		orders:  orders,
		coupons: coupons,
		events:  events,
	}
}

func (uc *ApplyDiscountUseCase) Execute(ctx context.Context, input ApplyDiscountInput) (*domain.Order, error) {
	ctx, span := tracer.Start(ctx, "ApplyDiscount")
	defer span.End()
	span.SetAttributes(
		attribute.String("order_id", input.OrderID.String()),
		attribute.String("coupon_code", input.CouponCode),
	)

	order, err := uc.orders.GetByID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	coupon, err := uc.coupons.GetByCode(ctx, input.CouponCode)
	if err != nil {
		return nil, err
	}

	if err := coupon.Validate(order.Subtotal); err != nil {
		return nil, err
	}

	discount := coupon.Calculate(order.Subtotal)
	if err := order.ApplyDiscount(discount, input.CouponCode); err != nil {
		return nil, err
	}

	if err := uc.orders.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("saving discount: %w", err)
	}

	if err := uc.coupons.IncrementUsage(ctx, input.CouponCode); err != nil {
		return nil, fmt.Errorf("incrementing coupon usage: %w", err)
	}

	evt := domain.NewOrderEvent(order)
	data, _ := json.Marshal(evt)
	_ = uc.events.Publish(ctx, "orders.discount_applied", data)

	return order, nil
}
