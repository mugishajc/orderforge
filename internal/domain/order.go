package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Order status lifecycle: Draft -> Confirmed -> Processing -> Shipped -> Delivered
// Cancellation is allowed from Draft, Confirmed, or Processing states.
type OrderStatus string

const (
	OrderStatusDraft      OrderStatus = "draft"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

var (
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrOrderNotFound      = errors.New("order not found")
	ErrProductNotFound    = errors.New("product not found")
	ErrInsufficientStock  = errors.New("insufficient stock for reservation")
	ErrInvalidQuantity    = errors.New("quantity must be greater than zero")
	ErrInvalidPrice       = errors.New("price must be non-negative")
	ErrEmptyOrder         = errors.New("order must contain at least one item")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrCouponNotFound     = errors.New("coupon not found")
	ErrCouponExpired      = errors.New("coupon has expired")
	ErrCouponMinNotMet    = errors.New("order total does not meet coupon minimum")
	ErrCouponAlreadyUsed  = errors.New("coupon usage limit reached")
	ErrDiscountExceedsMax = errors.New("discount exceeds maximum allowed")
)

// validTransitions defines which status transitions are permitted.
var validTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusDraft:      {OrderStatusConfirmed, OrderStatusCancelled},
	OrderStatusConfirmed:  {OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusProcessing: {OrderStatusShipped, OrderStatusCancelled},
	OrderStatusShipped:    {OrderStatusDelivered},
}

type Order struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Items       []OrderItem
	Status      OrderStatus
	Subtotal    int64 // in smallest currency unit (cents)
	DiscountAmt int64
	Total       int64
	CouponCode  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	UnitPrice int64 // cents
	LineTotal int64
}

// NewOrder creates a draft order with the given items.
// It validates quantities and calculates totals.
func NewOrder(customerID uuid.UUID, items []OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrEmptyOrder
	}

	orderID := uuid.New()
	now := time.Now().UTC()
	var subtotal int64

	for i := range items {
		if items[i].Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		if items[i].UnitPrice < 0 {
			return nil, ErrInvalidPrice
		}
		items[i].ID = uuid.New()
		items[i].OrderID = orderID
		items[i].LineTotal = int64(items[i].Quantity) * items[i].UnitPrice
		subtotal += items[i].LineTotal
	}

	return &Order{
		ID:         orderID,
		CustomerID: customerID,
		Items:      items,
		Status:     OrderStatusDraft,
		Subtotal:   subtotal,
		Total:      subtotal,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// TransitionTo moves the order to the target status if the transition is valid.
func (o *Order) TransitionTo(target OrderStatus) error {
	allowed, ok := validTransitions[o.Status]
	if !ok {
		return ErrInvalidTransition
	}
	for _, s := range allowed {
		if s == target {
			o.Status = target
			o.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrInvalidTransition
}

// ApplyDiscount subtracts the given amount from the order total.
// The discount cannot exceed the subtotal.
func (o *Order) ApplyDiscount(amount int64, couponCode string) error {
	if amount > o.Subtotal {
		return ErrDiscountExceedsMax
	}
	o.DiscountAmt = amount
	o.CouponCode = couponCode
	o.Total = o.Subtotal - amount
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// Recalculate recomputes subtotal and total from line items.
func (o *Order) Recalculate() {
	var subtotal int64
	for _, item := range o.Items {
		subtotal += item.LineTotal
	}
	o.Subtotal = subtotal
	o.Total = subtotal - o.DiscountAmt
	if o.Total < 0 {
		o.Total = 0
	}
}
