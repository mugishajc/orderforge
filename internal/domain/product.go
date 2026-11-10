package domain

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID        uuid.UUID
	Name      string
	SKU       string
	Price     int64 // cents
	Stock     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Reserve decrements stock by the requested quantity.
// Returns ErrInsufficientStock if stock is too low.
func (p *Product) Reserve(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if p.Stock < qty {
		return ErrInsufficientStock
	}
	p.Stock -= qty
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Release returns reserved stock back to available inventory.
func (p *Product) Release(qty int) {
	p.Stock += qty
	p.UpdatedAt = time.Now().UTC()
}
