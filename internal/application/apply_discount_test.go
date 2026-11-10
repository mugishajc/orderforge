package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/application"
	"github.com/mugishajc/orderforge/internal/domain"
)

func TestApplyDiscount_FixedCoupon(t *testing.T) {
	orderRepo := newMemOrderRepo()
	couponRepo := newMemCouponRepo()
	publisher := &recordingPublisher{}

	orderID := uuid.New()
	orderRepo.Create(context.Background(), &domain.Order{
		ID:       orderID,
		Status:   domain.OrderStatusDraft,
		Subtotal: 10000,
		Total:    10000,
	})

	couponRepo.Add(&domain.Coupon{
		Code:      "FLAT20",
		Type:      domain.DiscountTypeFixed,
		Value:     2000,
		ExpiresAt: time.Now().Add(48 * time.Hour),
	})

	uc := application.NewApplyDiscountUseCase(orderRepo, couponRepo, publisher)
	order, err := uc.Execute(context.Background(), application.ApplyDiscountInput{
		OrderID:    orderID,
		CouponCode: "FLAT20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Total != 8000 {
		t.Errorf("expected total 8000, got %d", order.Total)
	}
	if order.CouponCode != "FLAT20" {
		t.Errorf("expected coupon code FLAT20, got %s", order.CouponCode)
	}
}

func TestApplyDiscount_ExpiredCoupon(t *testing.T) {
	orderRepo := newMemOrderRepo()
	couponRepo := newMemCouponRepo()

	orderID := uuid.New()
	orderRepo.Create(context.Background(), &domain.Order{
		ID:       orderID,
		Status:   domain.OrderStatusDraft,
		Subtotal: 10000,
		Total:    10000,
	})

	couponRepo.Add(&domain.Coupon{
		Code:      "EXPIRED",
		Type:      domain.DiscountTypeFixed,
		Value:     1000,
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	})

	uc := application.NewApplyDiscountUseCase(orderRepo, couponRepo, &recordingPublisher{})
	_, err := uc.Execute(context.Background(), application.ApplyDiscountInput{
		OrderID:    orderID,
		CouponCode: "EXPIRED",
	})
	if err != domain.ErrCouponExpired {
		t.Errorf("expected ErrCouponExpired, got %v", err)
	}
}
