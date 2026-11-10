package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
)

func TestNewOrder_ValidItems(t *testing.T) {
	items := []domain.OrderItem{
		{ProductID: uuid.New(), Quantity: 2, UnitPrice: 1500},
		{ProductID: uuid.New(), Quantity: 1, UnitPrice: 3000},
	}

	order, err := domain.NewOrder(uuid.New(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Status != domain.OrderStatusDraft {
		t.Errorf("expected status draft, got %s", order.Status)
	}
	if order.Subtotal != 6000 {
		t.Errorf("expected subtotal 6000, got %d", order.Subtotal)
	}
	if order.Total != 6000 {
		t.Errorf("expected total 6000, got %d", order.Total)
	}
	if len(order.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(order.Items))
	}
}

func TestNewOrder_EmptyItems(t *testing.T) {
	_, err := domain.NewOrder(uuid.New(), nil)
	if err != domain.ErrEmptyOrder {
		t.Errorf("expected ErrEmptyOrder, got %v", err)
	}
}

func TestNewOrder_InvalidQuantity(t *testing.T) {
	items := []domain.OrderItem{
		{ProductID: uuid.New(), Quantity: 0, UnitPrice: 1000},
	}
	_, err := domain.NewOrder(uuid.New(), items)
	if err != domain.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestNewOrder_NegativePrice(t *testing.T) {
	items := []domain.OrderItem{
		{ProductID: uuid.New(), Quantity: 1, UnitPrice: -500},
	}
	_, err := domain.NewOrder(uuid.New(), items)
	if err != domain.ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestOrder_TransitionTo_Valid(t *testing.T) {
	cases := []struct {
		from domain.OrderStatus
		to   domain.OrderStatus
	}{
		{domain.OrderStatusDraft, domain.OrderStatusConfirmed},
		{domain.OrderStatusConfirmed, domain.OrderStatusProcessing},
		{domain.OrderStatusProcessing, domain.OrderStatusShipped},
		{domain.OrderStatusShipped, domain.OrderStatusDelivered},
		{domain.OrderStatusDraft, domain.OrderStatusCancelled},
		{domain.OrderStatusConfirmed, domain.OrderStatusCancelled},
		{domain.OrderStatusProcessing, domain.OrderStatusCancelled},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			order := &domain.Order{Status: tc.from}
			if err := order.TransitionTo(tc.to); err != nil {
				t.Errorf("expected valid transition, got %v", err)
			}
			if order.Status != tc.to {
				t.Errorf("expected status %s, got %s", tc.to, order.Status)
			}
		})
	}
}

func TestOrder_TransitionTo_Invalid(t *testing.T) {
	cases := []struct {
		from domain.OrderStatus
		to   domain.OrderStatus
	}{
		{domain.OrderStatusDelivered, domain.OrderStatusCancelled},
		{domain.OrderStatusCancelled, domain.OrderStatusDraft},
		{domain.OrderStatusShipped, domain.OrderStatusProcessing},
		{domain.OrderStatusDraft, domain.OrderStatusShipped},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			order := &domain.Order{Status: tc.from}
			if err := order.TransitionTo(tc.to); err != domain.ErrInvalidTransition {
				t.Errorf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}

func TestOrder_ApplyDiscount(t *testing.T) {
	order := &domain.Order{Subtotal: 10000, Total: 10000}

	err := order.ApplyDiscount(2000, "SAVE20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Total != 8000 {
		t.Errorf("expected total 8000, got %d", order.Total)
	}
	if order.CouponCode != "SAVE20" {
		t.Errorf("expected coupon SAVE20, got %s", order.CouponCode)
	}
}

func TestOrder_ApplyDiscount_ExceedsSubtotal(t *testing.T) {
	order := &domain.Order{Subtotal: 5000, Total: 5000}

	err := order.ApplyDiscount(6000, "TOOBIG")
	if err != domain.ErrDiscountExceedsMax {
		t.Errorf("expected ErrDiscountExceedsMax, got %v", err)
	}
}

func TestOrder_Recalculate(t *testing.T) {
	order := &domain.Order{
		Items: []domain.OrderItem{
			{LineTotal: 3000},
			{LineTotal: 4500},
		},
		DiscountAmt: 1000,
	}
	order.Recalculate()

	if order.Subtotal != 7500 {
		t.Errorf("expected subtotal 7500, got %d", order.Subtotal)
	}
	if order.Total != 6500 {
		t.Errorf("expected total 6500, got %d", order.Total)
	}
}

func TestProduct_Reserve(t *testing.T) {
	p := &domain.Product{Stock: 10}

	if err := p.Reserve(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Stock != 7 {
		t.Errorf("expected stock 7, got %d", p.Stock)
	}
}

func TestProduct_Reserve_Insufficient(t *testing.T) {
	p := &domain.Product{Stock: 2}

	err := p.Reserve(5)
	if err != domain.ErrInsufficientStock {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestProduct_Release(t *testing.T) {
	p := &domain.Product{Stock: 5}
	p.Release(3)
	if p.Stock != 8 {
		t.Errorf("expected stock 8, got %d", p.Stock)
	}
}

func TestCoupon_Validate_Expired(t *testing.T) {
	c := &domain.Coupon{
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	if err := c.Validate(10000); err != domain.ErrCouponExpired {
		t.Errorf("expected ErrCouponExpired, got %v", err)
	}
}

func TestCoupon_Validate_UsageLimitReached(t *testing.T) {
	c := &domain.Coupon{
		MaxUses:     5,
		CurrentUses: 5,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := c.Validate(10000); err != domain.ErrCouponAlreadyUsed {
		t.Errorf("expected ErrCouponAlreadyUsed, got %v", err)
	}
}

func TestCoupon_Validate_MinNotMet(t *testing.T) {
	c := &domain.Coupon{
		MinOrderAmt: 5000,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	if err := c.Validate(3000); err != domain.ErrCouponMinNotMet {
		t.Errorf("expected ErrCouponMinNotMet, got %v", err)
	}
}

func TestCoupon_Calculate_Fixed(t *testing.T) {
	c := &domain.Coupon{Type: domain.DiscountTypeFixed, Value: 2000}
	amt := c.Calculate(10000)
	if amt != 2000 {
		t.Errorf("expected 2000, got %d", amt)
	}
}

func TestCoupon_Calculate_Percent(t *testing.T) {
	c := &domain.Coupon{Type: domain.DiscountTypePercent, Value: 1500} // 15%
	amt := c.Calculate(10000)
	if amt != 1500 {
		t.Errorf("expected 1500, got %d", amt)
	}
}

func TestCoupon_Calculate_Fixed_CappedAtSubtotal(t *testing.T) {
	c := &domain.Coupon{Type: domain.DiscountTypeFixed, Value: 20000}
	amt := c.Calculate(5000)
	if amt != 5000 {
		t.Errorf("expected capped at 5000, got %d", amt)
	}
}
