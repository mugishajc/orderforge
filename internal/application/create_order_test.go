package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/application"
	"github.com/mugishajc/orderforge/internal/domain"
)

func TestCreateOrder_Success(t *testing.T) {
	orderRepo := newMemOrderRepo()
	productRepo := newMemProductRepo()
	publisher := &recordingPublisher{}

	pid := uuid.New()
	productRepo.Add(&domain.Product{
		ID:    pid,
		Name:  "Widget",
		SKU:   "WDG-001",
		Price: 2500,
		Stock: 10,
	})

	uc := application.NewCreateOrderUseCase(orderRepo, productRepo, &noopCache{}, publisher)

	order, err := uc.Execute(context.Background(), application.CreateOrderInput{
		CustomerID: uuid.New(),
		Items: []application.CreateOrderItemInput{
			{ProductID: pid, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Status != domain.OrderStatusDraft {
		t.Errorf("expected draft status, got %s", order.Status)
	}
	if order.Total != 7500 {
		t.Errorf("expected total 7500, got %d", order.Total)
	}

	// Verify stock was decremented
	p, _ := productRepo.GetByID(context.Background(), pid)
	if p.Stock != 7 {
		t.Errorf("expected remaining stock 7, got %d", p.Stock)
	}

	// Verify event was published
	if len(publisher.Messages) != 1 {
		t.Fatalf("expected 1 event, got %d", len(publisher.Messages))
	}
	if publisher.Messages[0].Subject != "orders.created" {
		t.Errorf("expected subject orders.created, got %s", publisher.Messages[0].Subject)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	orderRepo := newMemOrderRepo()
	productRepo := newMemProductRepo()
	publisher := &recordingPublisher{}

	pid := uuid.New()
	productRepo.Add(&domain.Product{
		ID:    pid,
		Name:  "Rare Item",
		SKU:   "RARE-001",
		Price: 5000,
		Stock: 2,
	})

	uc := application.NewCreateOrderUseCase(orderRepo, productRepo, &noopCache{}, publisher)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		CustomerID: uuid.New(),
		Items: []application.CreateOrderItemInput{
			{ProductID: pid, Quantity: 5},
		},
	})
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}

	// Stock should not have changed
	p, _ := productRepo.GetByID(context.Background(), pid)
	if p.Stock != 2 {
		t.Errorf("stock should remain 2 after failed order, got %d", p.Stock)
	}
}

func TestCreateOrder_ProductNotFound(t *testing.T) {
	uc := application.NewCreateOrderUseCase(
		newMemOrderRepo(), newMemProductRepo(), &noopCache{}, &recordingPublisher{},
	)

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		CustomerID: uuid.New(),
		Items: []application.CreateOrderItemInput{
			{ProductID: uuid.New(), Quantity: 1},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing product")
	}
}
