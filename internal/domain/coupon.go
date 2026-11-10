package domain

import (
	"time"
)

type DiscountType string

const (
	DiscountTypeFixed   DiscountType = "fixed"
	DiscountTypePercent DiscountType = "percent"
)

type Coupon struct {
	Code         string
	Type         DiscountType
	Value        int64 // cents for fixed, basis points for percent (e.g., 1000 = 10%)
	MinOrderAmt  int64 // minimum order subtotal in cents
	MaxUses      int
	CurrentUses  int
	ExpiresAt    time.Time
}

// Validate checks whether the coupon can be applied to an order with the given subtotal.
func (c *Coupon) Validate(orderSubtotal int64) error {
	if time.Now().UTC().After(c.ExpiresAt) {
		return ErrCouponExpired
	}
	if c.MaxUses > 0 && c.CurrentUses >= c.MaxUses {
		return ErrCouponAlreadyUsed
	}
	if orderSubtotal < c.MinOrderAmt {
		return ErrCouponMinNotMet
	}
	return nil
}

// Calculate returns the discount amount in cents for the given subtotal.
func (c *Coupon) Calculate(subtotal int64) int64 {
	switch c.Type {
	case DiscountTypeFixed:
		if c.Value > subtotal {
			return subtotal
		}
		return c.Value
	case DiscountTypePercent:
		// Value is in basis points: 1000 = 10%
		return subtotal * c.Value / 10000
	default:
		return 0
	}
}
