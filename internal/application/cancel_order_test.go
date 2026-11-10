package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/application"
	"github.com/mugishajc/orderforge/internal/domain"
)

func TestCancelOrder_Success(t *testing.T) {
	orderRepo := newMemOrderRepo()
	productRepo := newMemProductRepo()
	publisher := &recordingPublisher{}

	pid := uuid.New()
	productRepo.Add(&domain.Product{
		ID:    pid,
		Name:  "Gadget",
		SKU:   "GDG-001",
		Price: 3000,
		Stock: 7, // after reservation of 3
	})

	// Simulate an existing draft order
	orderID := uuid.New()
	orderRepo.Create(context.Background(), &domain.Order{
		ID:         orderID,
		CustomerID: uuid.New(),
		Status:     domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: pid, Quantity: 3, UnitPrice: 3000, LineTotal: 9000},
		},
		Subtotal: 9000,
		Total:    9000,
	})

	uc := application.NewCancelOrderUseCase(orderRepo, productRepo, &noopCache{}, publisher)
	err := uc.Execute(context.Background(), orderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status changed
	order, _ := orderRepo.GetByID(context.Background(), orderID)
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("expected cancelled, got %s", order.Status)
	}

	// Verify stock was released
	p, _ := productRepo.GetByID(context.Background(), pid)
	if p.Stock != 10 {
		t.Errorf("expected stock restored to 10, got %d", p.Stock)
	}

	// Verify event
	if len(publisher.Messages) != 1 || publisher.Messages[0].Subject != "orders.cancelled" {
		t.Error("expected orders.cancelled event")
	}
}

func TestCancelOrder_AlreadyShipped(t *testing.T) {
	orderRepo := newMemOrderRepo()
	publisher := &recordingPublisher{}

	orderID := uuid.New()
	orderRepo.Create(context.Background(), &domain.Order{
		ID:     orderID,
		Status: domain.OrderStatusShipped,
	})

	uc := application.NewCancelOrderUseCase(orderRepo, newMemProductRepo(), &noopCache{}, publisher)
	err := uc.Execute(context.Background(), orderID)
	if err == nil {
		t.Fatal("expected error cancelling shipped order")
	}
}
