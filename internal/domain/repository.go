package domain

import (
	"context"

	"github.com/google/uuid"
)

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	Update(ctx context.Context, order *Order) error
	ListByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*Order, error)
}

// ProductRepository defines persistence operations for products.
type ProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Update(ctx context.Context, product *Product) error
}

// CustomerRepository defines persistence operations for customers.
type CustomerRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
}

// CouponRepository defines persistence operations for coupons.
type CouponRepository interface {
	GetByCode(ctx context.Context, code string) (*Coupon, error)
	IncrementUsage(ctx context.Context, code string) error
}

// EventPublisher sends domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// ProductCache provides cached access to product data.
type ProductCache interface {
	Get(ctx context.Context, id uuid.UUID) (*Product, error)
	Set(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}
